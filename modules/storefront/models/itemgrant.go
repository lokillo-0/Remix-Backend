package models

type ItemGrant struct {
	TemplateId string                 `json:"templateId"`
	Quantity   int                    `json:"quantity"`
	Attributes map[string]interface{} `json:"attributes,omitempty"`
}
