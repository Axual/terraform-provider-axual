package webclient

type InstanceResponse struct {
	Name        string `json:"name"`
	ShortName   string `json:"shortName"`
	Description string `json:"description"`
	Uid         string `json:"uid"`
}
