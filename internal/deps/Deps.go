package deps

type Deps struct {
	Name     string    `bson:"name" json:"name"`
	System   string    `bson:"system" json:"system"`
	Versions []Version `bson:"versions" json:"versions"`
}

type Version struct {
	Version     string `bson:"version" json:"version"`
	PublishedAt string `bson:"publishedAt" json:"publishedAt"`
}
