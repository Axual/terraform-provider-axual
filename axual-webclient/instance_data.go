package webclient

type InstancesResponseByAttributes struct {
	Embedded struct {
		Instances []struct {
			Name        string `json:"name"`
			ShortName   string `json:"shortName"`
			Description string `json:"description"`
			Uid         string `json:"uid"`
		} `json:"instances"`
	} `json:"_embedded"`
}
