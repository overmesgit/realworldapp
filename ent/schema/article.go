package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// Article holds the schema definition for the Article entity.
type Article struct {
	ent.Schema
}

// Fields of the Article.
func (Article) Fields() []ent.Field {
	return []ent.Field{
		field.String("title"),
		field.Text("content"),
		field.Int("user_id"),
	}
}

// Edges of the Article.
func (Article) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("user", User.Type).
			Field("user_id").Required().Unique(),
	}
}
