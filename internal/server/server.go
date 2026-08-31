// Package server exposes a HomeBox instance as MCP tools.
//
// READ ONLY, deliberately. HomeBox's API also accepts POST on /entities and
// /tags, and PUT/DELETE on /tags/{id}; none of that is exposed here. A model
// that can rewrite an inventory it cannot see the consequences of is a
// different product, and a much more dangerous one.
package server

import (
	"context"
	"fmt"
	"net/url"
	"strconv"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/opwerm/homebox-mcp/internal/homebox"
)

// New builds the MCP server and registers every tool.
func New(c *homebox.Client, version string) *mcp.Server {
	s := mcp.NewServer(&mcp.Implementation{
		Name:    "homebox",
		Version: version,
	}, nil)

	// Tool names are prefixed so they read unambiguously when a client has
	// several MCP servers connected at once.

	type listArgs struct {
		Query     string   `json:"query,omitempty" jsonschema:"free-text search over names and descriptions"`
		Page      int      `json:"page,omitempty" jsonschema:"1-based page number"`
		PageSize  int      `json:"pageSize,omitempty" jsonschema:"results per page; HomeBox caps this itself"`
		Tags      []string `json:"tags,omitempty" jsonschema:"only entities carrying ALL of these tag ids"`
		ParentIDs []string `json:"parentIds,omitempty" jsonschema:"only entities directly inside these parent ids, e.g. a location"`
	}
	mcp.AddTool(s, &mcp.Tool{
		Name: "homebox_list_entities",
		// Says ITEMS, not "entities", because locations do not come back
		// here -- they are only reachable through the tree. Describing this
		// as covering both is worse than not having it: a model asked what
		// locations exist would call this, get nothing, and report there
		// are none.
		Description: "List or search ITEMS in the HomeBox inventory. Locations are NOT returned by this tool -- use homebox_entity_tree for those.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, a listArgs) (*mcp.CallToolResult, any, error) {
		q := url.Values{}
		if a.Query != "" {
			q.Set("q", a.Query)
		}
		if a.Page > 0 {
			q.Set("page", strconv.Itoa(a.Page))
		}
		if a.PageSize > 0 {
			q.Set("pageSize", strconv.Itoa(a.PageSize))
		}
		for _, t := range a.Tags {
			q.Add("tags", t)
		}
		for _, pid := range a.ParentIDs {
			q.Add("parentIds", pid)
		}

		return call(ctx, c, "/entities", q)
	})

	type idArgs struct {
		ID string `json:"id" jsonschema:"the entity UUID"`
	}
	mcp.AddTool(s, &mcp.Tool{
		Name:        "homebox_get_entity",
		Description: "Fetch one entity by id, with its full detail.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, a idArgs) (*mcp.CallToolResult, any, error) {
		if a.ID == "" {
			return nil, nil, fmt.Errorf("id is required")
		}

		return call(ctx, c, "/entities/"+url.PathEscape(a.ID), nil)
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "homebox_entity_tree",
		Description: "The location tree: how entities nest inside one another.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, any, error) {
		return call(ctx, c, "/entities/tree", nil)
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "homebox_entity_path",
		Description: "The full ancestor path of one entity, root first.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, a idArgs) (*mcp.CallToolResult, any, error) {
		if a.ID == "" {
			return nil, nil, fmt.Errorf("id is required")
		}

		return call(ctx, c, "/entities/"+url.PathEscape(a.ID)+"/path", nil)
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "homebox_list_tags",
		Description: "List tags. Tags replaced what older HomeBox called labels.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, any, error) {
		return call(ctx, c, "/tags", nil)
	})

	type assetArgs struct {
		AssetID string `json:"assetId" jsonschema:"the human-facing asset id, e.g. 000-001"`
	}
	mcp.AddTool(s, &mcp.Tool{
		Name:        "homebox_get_asset",
		Description: "Look an entity up by its printed asset id rather than its UUID.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, a assetArgs) (*mcp.CallToolResult, any, error) {
		if a.AssetID == "" {
			return nil, nil, fmt.Errorf("assetId is required")
		}

		return call(ctx, c, "/assets/"+url.PathEscape(a.AssetID), nil)
	})

	type statsArgs struct {
		Group string `json:"group,omitempty" jsonschema:"one of: locations, tags, purchase-price; omit for the overall totals"`
	}
	mcp.AddTool(s, &mcp.Tool{
		Name:        "homebox_statistics",
		Description: "Inventory statistics: totals overall, or broken down by locations, tags or purchase price.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, a statsArgs) (*mcp.CallToolResult, any, error) {
		path := "/groups/statistics"

		switch a.Group {
		case "":
		case "locations", "tags", "purchase-price":
			path += "/" + a.Group
		default:
			// Named explicitly rather than passed through: an arbitrary
			// value would become a path segment.
			return nil, nil, fmt.Errorf("group must be one of locations, tags, purchase-price")
		}

		return call(ctx, c, path, nil)
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "homebox_custom_fields",
		Description: "The custom field names defined on entities, for building queries.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, any, error) {
		return call(ctx, c, "/entities/fields", nil)
	})

	type fieldArgs struct {
		Field string `json:"field" jsonschema:"the custom field name, from homebox_custom_fields"`
	}
	mcp.AddTool(s, &mcp.Tool{
		Name:        "homebox_custom_field_values",
		Description: "The values in use for one custom field, so a query can filter on a real value rather than a guess.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, a fieldArgs) (*mcp.CallToolResult, any, error) {
		if a.Field == "" {
			return nil, nil, fmt.Errorf("field is required")
		}

		return call(ctx, c, "/entities/fields/values", url.Values{"field": {a.Field}})
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "homebox_get_tag",
		Description: "One tag by id.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, a idArgs) (*mcp.CallToolResult, any, error) {
		if a.ID == "" {
			return nil, nil, fmt.Errorf("id is required")
		}

		return call(ctx, c, "/tags/"+url.PathEscape(a.ID), nil)
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "homebox_entity_types",
		Description: "The entity types defined in this HomeBox, which distinguish an item from a location.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, any, error) {
		return call(ctx, c, "/entity-types", nil)
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "homebox_maintenance",
		Description: "Maintenance records across the inventory, or for one entity when an id is given.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, a idArgs) (*mcp.CallToolResult, any, error) {
		if a.ID != "" {
			return call(ctx, c, "/entities/"+url.PathEscape(a.ID)+"/maintenance", nil)
		}

		return call(ctx, c, "/maintenance", nil)
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "homebox_bill_of_materials",
		Description: "The bill of materials report for the whole inventory.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, any, error) {
		return call(ctx, c, "/reporting/bill-of-materials", nil)
	})

	return s
}

// call performs the request and returns the decoded JSON as structured
// content, so the client gets real data rather than a string it must parse.
func call(ctx context.Context, c *homebox.Client, path string, q url.Values) (*mcp.CallToolResult, any, error) {
	out, err := c.Raw(ctx, path, q)
	if err != nil {
		return nil, nil, err
	}

	if _, ok := out.(map[string]any); !ok {
		return nil, map[string]any{"items": out}, nil
	}

	return nil, out, nil
}
