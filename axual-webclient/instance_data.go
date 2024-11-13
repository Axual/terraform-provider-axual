package webclient

type InstanceResponse struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	ShortName   string `json:"shortName"`
	Uid         string `json:"uid"`
}
