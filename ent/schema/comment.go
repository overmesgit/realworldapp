package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// Comment holds the schema definition for the Comment entity.
type Comment struct {
	ent.Schema
}

// Fields of the Comment.
func (Comment) Fields() []ent.Field {
	return []ent.Field{
		field.String("text"),
		field.Int("article_id"),
		field.Int("user_id"),
	}
}

// Edges of the Comment.
func (Comment) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("article", Article.Type).
			Field("article_id").Required().Unique(),
		edge.To("user", User.Type).
			Field("user_id").Required().Unique(),
	}
}
