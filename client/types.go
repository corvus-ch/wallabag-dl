package client

import (
	"strings"
	"time"
)

type Entries struct {
	Page      int
	Limit     int
	Pages     int
	Total     int
	NaviLinks Links    `json:"_links"`
	Embedded  Embedded `json:"_embedded"`
}

type Embedded struct {
	Items []Item `json:"items"`
}

type Item struct {
	Links          Links        `json:"_links"`
	Annotations    []Annotation `json:"annotations"`
	CreatedAt      Time         `json:"created_at"`
	Content        string       `json:"content"`
	DomainName     string       `json:"domain_name"`
	ID             int          `json:"id"`
	IsArchived     int          `json:"is_archived"`
	IsStarred      int          `json:"is_starred"`
	Language       string       `json:"language"`
	Mimetype       string       `json:"mimetype"`
	PreviewPicture string       `json:"preview_picture"`
	ReadingTime    int          `json:"reading_time"`
	Tags           []Tag        `json:"tags"`
	Title          string       `json:"title"`
	UpdatedAt      Time         `json:"updated_at"`
	URL            string       `json:"url"`
	UserEmail      string       `json:"user_email"`
	UserID         int          `json:"user_id"`
	UserName       string       `json:"user_name"`
}

type Time struct {
	time.Time
}

func (t *Time) UnmarshalJSON(buf []byte) (err error) {
	s := strings.Trim(string(buf), `"`)
	t.Time, err = time.Parse("2006-01-02T15:04:05-0700", s)
	if err != nil {
		t.Time = time.Time{}
		return
	}
	return
}

type Links struct {
	Self  *Link
	First *Link
	Last  *Link
	Next  *Link
}

type Link struct {
	Href string
}

type Tag struct {
	ID    int    `json:"id"`
	Label string `json:"label"`
	Slug  string `json:"slug"`
}

type Annotations struct {
	Rows  []Annotation `json:"rows"`
	Total int          `json:"total"`
}

type Annotation struct {
	AnnotatorSchemaVersion string  `json:"annotator_schema_version"`
	CreatedAt              Time    `json:"created_at"`
	ID                     int     `json:"id"`
	Quote                  string  `json:"quote"`
	Ranges                 []Range `json:"ranges"`
	Text                   string  `json:"text"`
	UpdatedAt              Time    `json:"updated_at"`
}

type Range struct {
	End         string      `json:"end"`
	EndOffset   interface{} `json:"endOffset"`
	Start       string      `json:"start"`
	StartOffset interface{} `json:"startOffset"`
}
