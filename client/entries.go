package client

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
)

func (c *Client) GetEntries(param url.Values) ([]Item, error) {
	var entries Entries
	if err := c.Get(c.baseURL+"/api/entries.json?"+param.Encode(), &entries); err != nil {
		return nil, err
	}

	items := entries.Embedded.Items

	for entries.Page < entries.Pages {
		if err := c.Get(entries.NaviLinks.Next.Href, &entries); err != nil {
			return nil, err
		}

		items = append(items, entries.Embedded.Items...)
	}

	return items, nil
}

func (c *Client) GetEntry(id int) (Item, error) {
	var item Item
	err := c.Get(fmt.Sprintf("%s/api/entries/%d.json", c.baseURL, id), &item)

	return item, err
}

func (c *Client) ExportEntry(id int, format string, w io.Writer) error {
	resp, err := c.Request(http.MethodGet, fmt.Sprintf("%s/api/entries/%d/export.%s", c.baseURL, id, format), nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	_, err = io.Copy(w, resp.Body)

	return err
}
