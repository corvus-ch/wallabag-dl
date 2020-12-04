package client

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
)

func (c *Client) GetEntries(param url.Values) ([]Item, error) {
	log := c.log.WithValues("params", param.Encode())
	log.Info("Get entries")
	var entries Entries
	if err := c.Get(c.baseURL+"/api/entries.json?"+param.Encode(), &entries); err != nil {
		return nil, err
	}

	items := entries.Embedded.Items

	for entries.Page < entries.Pages {
		log.Info("Get entries from next page")
		if err := c.Get(entries.NaviLinks.Next.Href, &entries); err != nil {
			return nil, err
		}

		items = append(items, entries.Embedded.Items...)
	}

	log.Info("Retrieved entries", "retrieved", len(items), "total", entries.Total, "pages", entries.Pages)
	return items, nil
}

func (c *Client) GetEntry(id int) (Item, error) {
	c.log.Info("Get entry", "id", id)
	var item Item
	err := c.Get(fmt.Sprintf("%s/api/entries/%d.json", c.baseURL, id), &item)

	return item, err
}

func (c *Client) PatchEntry(id int, data map[string]interface{}) error {
	log := c.log.WithValues("id", id)
	for k, v := range data {
		log = log.WithValues(k, v)
	}
	log.Info("Patch entry")
	return c.Patch(fmt.Sprintf("%s/api/entries/%d.json", c.baseURL, id), data)
}

func (c *Client) ExportEntry(id int, format string, w io.Writer) error {
	c.log.Info("Export entry", "id", id, "format", format)
	resp, err := c.Request(http.MethodGet, fmt.Sprintf("%s/api/entries/%d/export.%s", c.baseURL, id, format), nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	_, err = io.Copy(w, resp.Body)

	return err
}
