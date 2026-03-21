package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j/dbtype"
	"github.com/orkestra/backend/internal/graph/models"
)

// GraphRepository defines the interface for Neo4j operations
type GraphRepository interface {
	ExecuteRead(ctx context.Context, database string, cypher string, params map[string]interface{}) (*models.QueryResult, error)
	ExecuteWrite(ctx context.Context, database string, cypher string, params map[string]interface{}) (*models.QueryResult, error)
	ListDatabases(ctx context.Context) ([]models.DatabaseInfo, error)
	GetSchema(ctx context.Context, database string) (*models.SchemaInfo, error)
	VerifyConnectivity(ctx context.Context) error
}

type neo4jRepository struct {
	driver          neo4j.DriverWithContext
	defaultDatabase string
}

// NewGraphRepository creates a new GraphRepository backed by Neo4j
func NewGraphRepository(driver neo4j.DriverWithContext, defaultDatabase string) GraphRepository {
	return &neo4jRepository{
		driver:          driver,
		defaultDatabase: defaultDatabase,
	}
}

func (r *neo4jRepository) resolveDatabase(database string) string {
	if database != "" {
		return database
	}
	return r.defaultDatabase
}

func (r *neo4jRepository) ExecuteRead(ctx context.Context, database string, cypher string, params map[string]interface{}) (*models.QueryResult, error) {
	db := r.resolveDatabase(database)
	session := r.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: db, AccessMode: neo4j.AccessModeRead})
	defer session.Close(ctx)

	start := time.Now()
	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		res, err := tx.Run(ctx, cypher, params)
		if err != nil {
			return nil, err
		}
		return collectResults(ctx, res)
	})
	if err != nil {
		return nil, fmt.Errorf("read query failed: %w", err)
	}

	qr := result.(*models.QueryResult)
	qr.Metadata.ExecutionTimeMs = time.Since(start).Milliseconds()
	return qr, nil
}

func (r *neo4jRepository) ExecuteWrite(ctx context.Context, database string, cypher string, params map[string]interface{}) (*models.QueryResult, error) {
	db := r.resolveDatabase(database)
	session := r.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: db, AccessMode: neo4j.AccessModeWrite})
	defer session.Close(ctx)

	start := time.Now()
	result, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		res, err := tx.Run(ctx, cypher, params)
		if err != nil {
			return nil, err
		}
		return collectResults(ctx, res)
	})
	if err != nil {
		return nil, fmt.Errorf("write query failed: %w", err)
	}

	qr := result.(*models.QueryResult)
	qr.Metadata.ExecutionTimeMs = time.Since(start).Milliseconds()
	return qr, nil
}

func (r *neo4jRepository) ListDatabases(ctx context.Context) ([]models.DatabaseInfo, error) {
	// SHOW DATABASES must run against the system database
	session := r.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: "system", AccessMode: neo4j.AccessModeRead})
	defer session.Close(ctx)

	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		res, err := tx.Run(ctx, "SHOW DATABASES", nil)
		if err != nil {
			return nil, err
		}

		var databases []models.DatabaseInfo
		for res.Next(ctx) {
			record := res.Record()
			db := models.DatabaseInfo{}

			if v, ok := record.Get("name"); ok {
				db.Name, _ = v.(string)
			}
			if v, ok := record.Get("address"); ok {
				db.Address, _ = v.(string)
			}
			if v, ok := record.Get("currentStatus"); ok {
				db.CurrentStatus, _ = v.(string)
			}
			if v, ok := record.Get("default"); ok {
				db.Default, _ = v.(bool)
			}
			if v, ok := record.Get("home"); ok {
				db.Home, _ = v.(bool)
			}
			databases = append(databases, db)
		}
		if err := res.Err(); err != nil {
			return nil, err
		}
		return databases, nil
	})
	if err != nil {
		return nil, fmt.Errorf("list databases failed: %w", err)
	}

	return result.([]models.DatabaseInfo), nil
}

func (r *neo4jRepository) GetSchema(ctx context.Context, database string) (*models.SchemaInfo, error) {
	db := r.resolveDatabase(database)
	session := r.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: db, AccessMode: neo4j.AccessModeRead})
	defer session.Close(ctx)

	schema := &models.SchemaInfo{}

	_, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		// Get labels with counts
		res, err := tx.Run(ctx, "CALL db.labels() YIELD label RETURN label", nil)
		if err != nil {
			return nil, err
		}
		for res.Next(ctx) {
			name, _ := res.Record().Get("label")
			label := models.LabelInfo{Name: name.(string)}

			// Get count for this label
			countRes, err := tx.Run(ctx, fmt.Sprintf("MATCH (n:`%s`) RETURN count(n) AS cnt", name.(string)), nil)
			if err == nil && countRes.Next(ctx) {
				if cnt, ok := countRes.Record().Get("cnt"); ok {
					label.Count, _ = cnt.(int64)
				}
			}

			// Get properties for this label
			propRes, err := tx.Run(ctx, fmt.Sprintf("MATCH (n:`%s`) UNWIND keys(n) AS key RETURN DISTINCT key", name.(string)), nil)
			if err == nil {
				for propRes.Next(ctx) {
					if key, ok := propRes.Record().Get("key"); ok {
						label.Properties = append(label.Properties, key.(string))
					}
				}
			}
			schema.Labels = append(schema.Labels, label)
		}

		// Get relationship types with counts
		res, err = tx.Run(ctx, "CALL db.relationshipTypes() YIELD relationshipType RETURN relationshipType", nil)
		if err != nil {
			return nil, err
		}
		for res.Next(ctx) {
			name, _ := res.Record().Get("relationshipType")
			relType := models.RelTypeInfo{Name: name.(string)}

			countRes, err := tx.Run(ctx, fmt.Sprintf("MATCH ()-[r:`%s`]->() RETURN count(r) AS cnt", name.(string)), nil)
			if err == nil && countRes.Next(ctx) {
				if cnt, ok := countRes.Record().Get("cnt"); ok {
					relType.Count, _ = cnt.(int64)
				}
			}

			propRes, err := tx.Run(ctx, fmt.Sprintf("MATCH ()-[r:`%s`]->() UNWIND keys(r) AS key RETURN DISTINCT key", name.(string)), nil)
			if err == nil {
				for propRes.Next(ctx) {
					if key, ok := propRes.Record().Get("key"); ok {
						relType.Properties = append(relType.Properties, key.(string))
					}
				}
			}
			schema.RelationshipTypes = append(schema.RelationshipTypes, relType)
		}

		// Get indexes
		res, err = tx.Run(ctx, "SHOW INDEXES", nil)
		if err == nil {
			for res.Next(ctx) {
				record := res.Record()
				idx := models.IndexInfo{}
				if v, ok := record.Get("name"); ok {
					idx.Name, _ = v.(string)
				}
				if v, ok := record.Get("type"); ok {
					idx.Type, _ = v.(string)
				}
				if v, ok := record.Get("entityType"); ok {
					idx.EntityType, _ = v.(string)
				}
				if v, ok := record.Get("labelsOrTypes"); ok {
					idx.Labels = toStringSlice(v)
				}
				if v, ok := record.Get("properties"); ok {
					idx.Properties = toStringSlice(v)
				}
				if v, ok := record.Get("state"); ok {
					idx.State, _ = v.(string)
				}
				schema.Indexes = append(schema.Indexes, idx)
			}
		}

		// Get constraints
		res, err = tx.Run(ctx, "SHOW CONSTRAINTS", nil)
		if err == nil {
			for res.Next(ctx) {
				record := res.Record()
				c := models.ConstraintInfo{}
				if v, ok := record.Get("name"); ok {
					c.Name, _ = v.(string)
				}
				if v, ok := record.Get("type"); ok {
					c.Type, _ = v.(string)
				}
				if v, ok := record.Get("entityType"); ok {
					c.EntityType, _ = v.(string)
				}
				if v, ok := record.Get("labelsOrTypes"); ok {
					c.Labels = toStringSlice(v)
				}
				if v, ok := record.Get("properties"); ok {
					c.Properties = toStringSlice(v)
				}
				schema.Constraints = append(schema.Constraints, c)
			}
		}

		// Get total counts
		countRes, err := tx.Run(ctx, "MATCH (n) RETURN count(n) AS nodeCount", nil)
		if err == nil && countRes.Next(ctx) {
			if cnt, ok := countRes.Record().Get("nodeCount"); ok {
				schema.NodeCount, _ = cnt.(int64)
			}
		}

		countRes, err = tx.Run(ctx, "MATCH ()-[r]->() RETURN count(r) AS relCount", nil)
		if err == nil && countRes.Next(ctx) {
			if cnt, ok := countRes.Record().Get("relCount"); ok {
				schema.RelationshipCount, _ = cnt.(int64)
			}
		}

		return nil, nil
	})

	return schema, err
}

func (r *neo4jRepository) VerifyConnectivity(ctx context.Context) error {
	return r.driver.VerifyConnectivity(ctx)
}

// collectResults processes a Neo4j result set into a QueryResult
func collectResults(ctx context.Context, res neo4j.ResultWithContext) (*models.QueryResult, error) {
	qr := &models.QueryResult{
		Graph: &models.GraphData{},
	}

	// Track unique nodes/rels for graph extraction
	seenNodes := make(map[int64]bool)
	seenRels := make(map[int64]bool)

	keys, err := res.Keys()
	if err != nil {
		return nil, err
	}
	qr.Columns = keys

	for res.Next(ctx) {
		record := res.Record()
		row := make(map[string]interface{})

		for _, key := range keys {
			val, _ := record.Get(key)
			row[key] = marshalValue(val, qr.Graph, seenNodes, seenRels)
		}
		qr.Rows = append(qr.Rows, row)
	}

	if err := res.Err(); err != nil {
		return nil, err
	}

	// Extract counters from summary
	summary, err := res.Consume(ctx)
	if err == nil && summary.Counters().ContainsUpdates() {
		c := summary.Counters()
		qr.Metadata.NodesCreated = c.NodesCreated()
		qr.Metadata.NodesDeleted = c.NodesDeleted()
		qr.Metadata.RelationshipsCreated = c.RelationshipsCreated()
		qr.Metadata.RelationshipsDeleted = c.RelationshipsDeleted()
		qr.Metadata.PropertiesSet = c.PropertiesSet()
		qr.Metadata.LabelsAdded = c.LabelsAdded()
		qr.Metadata.LabelsRemoved = c.LabelsRemoved()
		qr.Metadata.ContainsUpdates = true
	}

	qr.Metadata.ResultCount = len(qr.Rows)

	// Clean up empty graph data
	if len(qr.Graph.Nodes) == 0 && len(qr.Graph.Relationships) == 0 {
		qr.Graph = nil
	}

	return qr, nil
}

// marshalValue converts Neo4j driver types to JSON-serializable values
// and extracts graph elements for visualization
func marshalValue(val interface{}, graph *models.GraphData, seenNodes map[int64]bool, seenRels map[int64]bool) interface{} {
	if val == nil {
		return nil
	}

	switch v := val.(type) {
	case dbtype.Node:
		node := models.GraphNode{
			ID:         v.Id,
			ElementID:  v.ElementId,
			Labels:     v.Labels,
			Properties: sanitizeProps(v.Props),
		}
		if graph != nil && !seenNodes[v.Id] {
			seenNodes[v.Id] = true
			graph.Nodes = append(graph.Nodes, node)
		}
		return node

	case dbtype.Relationship:
		rel := models.GraphRelationship{
			ID:             v.Id,
			ElementID:      v.ElementId,
			Type:           v.Type,
			StartNodeID:    v.StartId,
			EndNodeID:      v.EndId,
			StartElementID: v.StartElementId,
			EndElementID:   v.EndElementId,
			Properties:     sanitizeProps(v.Props),
		}
		if graph != nil && !seenRels[v.Id] {
			seenRels[v.Id] = true
			graph.Relationships = append(graph.Relationships, rel)
		}
		return rel

	case dbtype.Path:
		// Extract all nodes and relationships from the path
		pathData := map[string]interface{}{
			"_type": "path",
		}
		var nodes []interface{}
		for _, n := range v.Nodes {
			nodes = append(nodes, marshalValue(n, graph, seenNodes, seenRels))
		}
		pathData["nodes"] = nodes

		var rels []interface{}
		for _, r := range v.Relationships {
			rels = append(rels, marshalValue(r, graph, seenNodes, seenRels))
		}
		pathData["relationships"] = rels
		return pathData

	case dbtype.Point2D:
		return map[string]interface{}{
			"_type": "point2d",
			"srid":  v.SpatialRefId,
			"x":     v.X,
			"y":     v.Y,
		}

	case dbtype.Point3D:
		return map[string]interface{}{
			"_type": "point3d",
			"srid":  v.SpatialRefId,
			"x":     v.X,
			"y":     v.Y,
			"z":     v.Z,
		}

	case []interface{}:
		result := make([]interface{}, len(v))
		for i, item := range v {
			result[i] = marshalValue(item, graph, seenNodes, seenRels)
		}
		return result

	case map[string]interface{}:
		result := make(map[string]interface{})
		for k, item := range v {
			result[k] = marshalValue(item, graph, seenNodes, seenRels)
		}
		return result

	default:
		return v
	}
}

// sanitizeProps converts Neo4j property maps to clean maps, handling special types
func sanitizeProps(props map[string]interface{}) map[string]interface{} {
	if props == nil {
		return make(map[string]interface{})
	}
	result := make(map[string]interface{}, len(props))
	for k, v := range props {
		result[k] = marshalValue(v, nil, nil, nil)
	}
	return result
}

// toStringSlice converts an interface{} to a string slice
func toStringSlice(val interface{}) []string {
	if val == nil {
		return nil
	}
	switch v := val.(type) {
	case []interface{}:
		result := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				result = append(result, s)
			}
		}
		return result
	case []string:
		return v
	default:
		return nil
	}
}
