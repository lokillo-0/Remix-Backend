package mcp

type BaseItem struct {
	Attributes interface{} `json:"attributes"`
	TemplateId string      `json:"templateId"`
	Quantity   int         `json:"quantity"`
}
