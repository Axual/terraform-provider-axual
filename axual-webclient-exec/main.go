package main

/*
	For manual testing of Axual's golang webclient in axual-webclient.
	For usage please change credentials in getClient().
*/
import (
	webclient "axual-webclient"
	"log"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	c := getClient()

	/*
		Find ApplicationAccessGrant by application and environment
	*/

	accessGrantRequest := webclient.ApplicationAccessGrantAttributes{
		//TopicId:       "qq1",
		ApplicationId: "4502372a6c894fa98dfa233f3c68b9d0",
		EnvironmentId: "48684b9c0da74f5da4d3249b769f8536",
		Statuses:      "APPROVED",
	}
	applicationAccessGrant, err := c.GetApplicationAccessGrantsByAttributes(accessGrantRequest)
	if err != nil {
		return
	}
	if len(applicationAccessGrant.Embedded.ApplicationAccessGrantResponses) > 0 {
		log.Printf("FOUND APPLICATION PRINCIPAL BY APPLICATION AND ENVIRONMENT: %s\n", applicationAccessGrant.Embedded.ApplicationAccessGrantResponses[0])
	} else {
		log.Println("NO APPLICATION PRINCIPAL FOUND FOR THE GIVEN APPLICATION AND ENVIRONMENT")
	}

	/*
		Find ApplicationDeployment by application and environment
	*/
	//applicationPrincipalFindByApplicationAndEnvironmentResponse, err :=
	//	c.FindApplicationPrincipalByApplicationAndEnvironment("https://self-service.qa.np.westeurope.azure.axual.cloud/api/applications/4502372a6c894fa98dfa233f3c68b9d0",
	//		"https://self-service.qa.np.westeurope.azure.axual.cloud/api/environments/48684b9c0da74f5da4d3249b769f8536")
	//if err != nil {
	//	return
	//}
	//if len(applicationPrincipalFindByApplicationAndEnvironmentResponse.Embedded.ApplicationPrincipalResponses) > 0 {
	//	log.Printf("FOUND APPLICATION PRINCIPAL BY APPLICATION AND ENVIRONMENT: %s\n", applicationPrincipalFindByApplicationAndEnvironmentResponse.Embedded.ApplicationPrincipalResponses[0])
	//} else {
	//	log.Println("NO APPLICATION PRINCIPAL FOUND FOR THE GIVEN APPLICATION AND ENVIRONMENT")
	//}

	//log.Println("APPLICATION DEPLOYMENT IS", applicationDeployment.Configs)

	//log.Printf("FOUND APPLICATION PRINCIPAL BY APPLICATION AND ENVIRONMENT: %s\n", applicationPrincipalFindByApplicationAndEnvironmentResponse.Embedded.ApplicationPrincipalResponses[0])

	///*
	//	STATUS Application Deployment
	//*/
	//applicationDeploymentStatus, _ := c.GetApplicationDeploymentStatus("ba731ec319c740b7a18fc9d86992b385")
	//log.Printf("STATUS APPLICATION: %s\n", applicationDeploymentStatus)

	/////*
	////	STOP Application Deployment
	////*/
	//var applicationStartRequest = webclient.ApplicationDeploymentOperationRequest{
	//	Action: "STOP",
	//}
	//application := c.OperateApplicationDeployment("97967dfecf5b49c18b4d2d4b83dce3d2", "STOP", applicationStartRequest)
	//log.Printf("STOP APPLICATION: %s\n", application)

	/////*
	////	START Application Deployment
	////*/
	//var applicationStartRequest = webclient.ApplicationDeploymentOperationRequest{
	//	Action: "START",
	//}
	//application := c.OperateApplicationDeployment("97967dfecf5b49c18b4d2d4b83dce3d2", "START", applicationStartRequest)
	//log.Printf("START APPLICATION: %s\n", application)

	/*
		Create Connector Application Principal
	*/

	//var array [1]webclient.ApplicationPrincipalRequest
	//array[0] = webclient.ApplicationPrincipalRequest{
	//	Principal:   "-----BEGIN CERTIFICATE-----\nMIIEsjCCApqgAwIBAgIRANlGfD2h/5K+2AqXk+2/GmswDQYJKoZIhvcNAQELBQAw\nKzEpMCcGA1UEAwwgQXh1YWwgRHVtbXkgSW50ZXJtZWRpYXRlIDIwMTggMDEwHhcN\nMjAwNDAxMTIzMzUzWhcNMjUwMzMxMTIzMzUzWjBXMQswCQYDVQQGEwJOTDEQMA4G\nA1UEBwwHVXRyZWNodDETMBEGA1UECgwKQXh1YWwgQi5WLjEhMB8GA1UEAwwYRXhh\nbXBsZSBTdHJlYW0gUHJvY2Vzc29yMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIB\nCgKCAQEAo8DILU6LFXEpN2DcD7f9haAMq9ddlDYDomDM7PAk7UwYGUuycpX11z9s\nSdbO5a0WNc/qN2Zi/tPn7cYrYod6nsLG+NfBSMYNakvc7at8Ym2m2I3k5klQM3YH\nSTUf+a9O3Gfz6Q2XZ6GvHIf63omEH8cBD9fk9BLmDKgQHwcNf9mvWa4se9ONsY1l\nGs/R+q7hmGLZbbyr2lWSrXp0GKKfoxtS/mv0Z0lcuvxgGpB3HHMY6pegBuhOOVwC\nv1HUUTAORoYmwOuFD0fO6vAmwGUuyPwCSJxcDXGJYwnpK953yJYM1UAF/BKe5fFM\nZ3Ce75R06feQaZYilp7XQN4YYBJWZwIDAQABo4GkMIGhMAkGA1UdEwQCMAAwHQYD\nVR0OBBYEFLjtlM29zjwQ7SR5IbR3CARf+ENiMEkGA1UdIwRCMECAFGvSKVeLUjYs\nBQrGV7kadHisEx9moSSkIjAgMR4wHAYDVQQDDBVBeHVhbCBEdW1teSBSb290IDIw\nMTiCAhAAMAsGA1UdDwQEAwIEsDAdBgNVHSUEFjAUBggrBgEFBQcDAgYIKwYBBQUH\nAwEwDQYJKoZIhvcNAQELBQADggIBAKk6ID1SYackve9eCe9hk//OZY7wWUirRP+7\nJIoCi+cAMx4hn1M5XPi4yQaGMe2vQXgcOgu2h7ljqXdpkSL4ZTAv1PCyOEMHJ0Wn\nHc4biKr6reQ7GQzsjygB5ULr9T5CEH/HAfAw9n7ttrxadPOlLbgilucyO8LeLrPZ\nJ0Jbo1Yf1VJE5SZmWN9U47AlDhEZnWjAQolcIqVWOcVYYT12ca9epH+slnBXDX7B\ny41AYaPwA0c/j+mlSXdFsU8HxU9NSu1R15bzW3+4uTIPvIeB3s+yjHBt35Sb9IRS\nGoaGIcE9HJZh9QTqgPUDxlXgDvWttfzIp2wNeWpmd9WP+Ai7CT9M2PxSGYDunl27\neY/U83AaScaSoSXn9tXHtXyRHrD/OAlWc+NzZtFJxRlRw7sc4ElmHbSoBVcDpi38\nTGBD+zxTpKMeTaVhJpGvhzc9MVy5qaI6Bcdg6J4SusiTiMro9bu/kxuxrLcE1E7w\nAmeqQwPoaZZ3O200kd4bd7Iil6g3mP8nVkkeLuJA5/ax9FEdbEH+jpfk3ouf+AKe\np1whWsEx5KOqApnl6NByRq+zqohofmyDWCIXzSXp0neQEryByatZDUVuf1HUQCAv\nFkoZdzOnT+scywMfxyqrP0N8nvh/0CBPX/s+7B9v+KJm54ZQcxAEsms/NvkNPunx\nVANhOECv\n-----END CERTIFICATE-----\n-----BEGIN CERTIFICATE-----\nMIIFLTCCAxWgAwIBAgICEAAwDQYJKoZIhvcNAQELBQAwIDEeMBwGA1UEAwwVQXh1\nYWwgRHVtbXkgUm9vdCAyMDE4MB4XDTE4MDUyOTExMDEzNFoXDTI4MDUyNjExMDEz\nNFowKzEpMCcGA1UEAwwgQXh1YWwgRHVtbXkgSW50ZXJtZWRpYXRlIDIwMTggMDEw\nggIiMA0GCSqGSIb3DQEBAQUAA4ICDwAwggIKAoICAQC9uOuzJekmeo3hl8fjQlKS\nHApS3llcliq1YrXpkMbHAA9StHaMHPW+Dzr2/+cdfBAmN3sujCY8Paq15QI+TDOq\nKA5SByCBQKXx2qulBPcZs3mDMt+KxAaeWfwR4Nj0NNKbmw2HjDddo77joeVOuOX2\n4o1wXzmAAolVMIcRYA11EMWNUtYrHCzBa7RfYht2G5dE69ckrgfw1Nxs01Sbg+xP\nsK9aK/LHPUalYZNY+76x7vabEpzaPfpyKzDTWA20SPk0WfTf9/+K3o+urzDG8O/q\nw9xbBOzWohGmRyA/z841p1SD7inpZcyO/KeW1yTP2WyFxADwUrv2mEYXnma/Gdna\nG62IQYk/UMex9W8pT6tfwrg/36sSwr88yPR5dJxzjHUE+w/rYG3k+K+EqvZ5qOC5\n32AJ9BS2nbNuGpmRU1qoMCwpL7B2E/CKJLIdFcf/qmcnWJEXo+u34+fQZg8XaDCI\nXhUqAHz6YkjCiFGd/JwL1IqsfxFsV9wHTUbW2AumglU65ZrjhXrrzE7Hk9ng1spJ\ndOwfBihBNjnr0mKHY9leJ3chJ9HQ55/fEgcRNrj8EC69QCeAtpY5yOAjKpA03UvF\ngrDt8CIyIehNUwTXIhQSHZU4eZ0rzWf0vvMbhL2FvKtphbpnNKoXeNLv2IMZpT4B\nVwsqLqaIkl/I4FPpYBoSYwIDAQABo2YwZDAdBgNVHQ4EFgQUa9IpV4tSNiwFCsZX\nuRp0eKwTH2YwHwYDVR0jBBgwFoAUdKOPDqSFQ6Bfk0I/asBkByt5gsUwEgYDVR0T\nAQH/BAgwBgEB/wIBADAOBgNVHQ8BAf8EBAMCAYYwDQYJKoZIhvcNAQELBQADggIB\nAKoNIqiOdjlUBNg7cvR9Su4KgGrsZf78pG1H2MlNxJjFM/80EiWEfze/EG2MLxFq\n8vToIHDjb0kVetYpdmfHNXTTlaaroBlXwyUYToPzQ985qr3LD8RhYZFAsiZCTtpJ\n4FT6sh/mccTyx8G8+ZS6mn/le2WPj/t6beNLgbdl5n8fghdQcmT/TqGXE50UftWt\nHSx3fsq2aKuNdVzhKzTin50IbiE9DV1dKo6B+ipOy/Dz5GMv3Z/3ntLTvxabCMOl\n7s7WsUE7VPABRSifUS80Z9Ai38faLSu+Ouzx40ceXwvlFQtJ2LYQ8Ru5Q63k2wB3\nEOE6cgAhiYExrz3fDDtUkui9vIfWfTPMnXR7xQ8YqK4Qqld2ESxvMQU2jzbZKSf+\n3sWnPvN4HTg0cfysmOdLGZwf3u8A9tMtxhUEtxUx7r76M4ekSKdNv1Nf5u5N/h7b\nAbEqSp1XADTxkE448i7hNJzn2Ce6JtFya231Ni0xyYKQIajP18jNypAw1eABYFkN\n53vQTUfqcbtcrCios1xRdDqfgkYaKZv7p63aoObFTf/mmG7sFjGAEPQscagOukwN\nwnkjCVifVbk5qJUaUWSLeYziI+HYkEA9P/h4o83nbf0YgBtOFoc0XWKmKagHifZN\nSEJ9kRCWzYaL2ChiL6jHGh26WT/hbNKeAlcxPnT4u/l1\n-----END CERTIFICATE-----\n-----BEGIN CERTIFICATE-----\nMIIFJjCCAw6gAwIBAgIJAINuAirfnRU6MA0GCSqGSIb3DQEBCwUAMCAxHjAcBgNV\nBAMMFUF4dWFsIER1bW15IFJvb3QgMjAxODAeFw0xODA1MjkxMDM0MTRaFw0zODA1\nMjQxMDM0MTRaMCAxHjAcBgNVBAMMFUF4dWFsIER1bW15IFJvb3QgMjAxODCCAiIw\nDQYJKoZIhvcNAQEBBQADggIPADCCAgoCggIBAMVDjbhq3TGuQ6INTZ+dhSIgsdbq\nw2nxF3myrS7v89bcNxMyLypWYTmR4OAYRXRBnW4KX6sTubPyL3ogPz6hXmfmPfAz\n+X//HTIiybL3e3qwxqWphp09+JT6veEp/e/wEEjSMj5nsxkDEjj9JEQWu/1B+N+V\nXOJkTYFy05ZgeWplkyLwT71myF047aISK27a+VebBMaPpvvetScbMSwxAbk51cGV\nUC4gpwvnvsbp/CRuMV0dYzkeTmxgn860l3s8+7qUJoOrtiO0cDpv97SK9Ck9ef1k\nR6KFttzxb/u+eMFi3RUErEGwE8P3thTseXRkp5hMwcyaSQv0wfLawlwcNFGOzsBx\nfJS7QUIUpEyzRqj5Ppgaj530APxbgitLOfVLZ2DvcBcmnQns6OE+uwymuvAj8Ftj\n6AFJXH2lmswHLl5uD9kIOwmpZg4NZLP2Qv+WOT6HLgI7Kv1z0OV2H7UlWA7hwQXl\noQ6fJ2YLEhT+GM9xHKJ+DQCxvjWvtGUSb/Dk0j/R9mpSFfHvVJgE/xV+7F7Vlyw5\n/cDpF3GZOTGQ/MFy4RqRrTtjnZw2/bZZyJ+Xb743OeQhABFUdadh8cmyehDregtr\nalHxtjKxCxrT55OHCYhbCoz6nEnQURD7EPQhU5puUKalRq2ApDkveIk8uj0HQmQm\nKyRuNX7M6vCoWnpxAgMBAAGjYzBhMB0GA1UdDgQWBBR0o48OpIVDoF+TQj9qwGQH\nK3mCxTAfBgNVHSMEGDAWgBR0o48OpIVDoF+TQj9qwGQHK3mCxTAPBgNVHRMBAf8E\nBTADAQH/MA4GA1UdDwEB/wQEAwIBhjANBgkqhkiG9w0BAQsFAAOCAgEAbJanqR4P\nmr05AyAu8vlrLsleXA8VAPDiaaYStYH5cIdBBWkaIxanLFDmbyQwKkKdkHQWV9X8\n1P52q49T9RsoBsEOmwdiaCY2PEUz7Y3bFW0UeM+k65VlHlXWywRM6+O02t4TrJXH\nF6h7vPon01OwhgW9Yil/Kr+yyZK50Ic+pm4UhHmtxY932cNaRCdae5tKsjabsP7Z\nrdAksLia8mTp+HADkZJ1uODxyDh0S1WMKB5JoHYBrmtUr1NYLgRC6SinhK4r7rbi\nEWuurE605Nm//jv3Czdy8gEsMDtXLZYY0iqGnD11MAJFXyQ6PG2eq1cXcsJNRojm\n8D4ipfQ+z4bp9dDVR2DzVyTYe4yuhZuIe2phOhPc8KkBaXQRMHfVKyeEmzqEFLaM\nkfaDZkRsrMZSqh+KJoxDG3h8UqssChX+cuZdsjRhNWRqfbB20I9Upwa+XooyCU4E\nEkYyFTMchtvbYZEN/XvlPfhK5JB9eJ5rrcE8hKsP3gftchWWqCDedKugvZW/t5Vk\nlc+z4IjiJFnRDfcr4Z5V2Hpseyno3AEK7aUdJlmuPnxoImFXfQ4jUguM/wznJHl7\nXv9T0oaBVHM7Bd6PlES04Oho0KZXS6NryTsZn9GFV4qGZj5lEeOVl15AOfeIjP/I\nokA2uUH/ZuJlR/BEmqbLt5HWPRNT/GgLfPY=\n-----END CERTIFICATE-----",
	//	PrivateKey:  "-----BEGIN PRIVATE KEY-----\nMIIEvQIBADANBgkqhkiG9w0BAQEFAASCBKcwggSjAgEAAoIBAQCjwMgtTosVcSk3\nYNwPt/2FoAyr112UNgOiYMzs8CTtTBgZS7JylfXXP2xJ1s7lrRY1z+o3ZmL+0+ft\nxitih3qewsb418FIxg1qS9ztq3xibabYjeTmSVAzdgdJNR/5r07cZ/PpDZdnoa8c\nh/reiYQfxwEP1+T0EuYMqBAfBw1/2a9Zrix7042xjWUaz9H6ruGYYtltvKvaVZKt\nenQYop+jG1L+a/RnSVy6/GAakHcccxjql6AG6E45XAK/UdRRMA5GhibA64UPR87q\n8CbAZS7I/AJInFwNcYljCekr3nfIlgzVQAX8Ep7l8UxncJ7vlHTp95BpliKWntdA\n3hhgElZnAgMBAAECggEAT/x+ywKulU6tH0vEsjjhMUA4cEfGgfrQU6SM9L9oijC9\nJL4A8E+nwfhM4NR4Q2AElUx1z5HHkUnY5CAEWSBsGNhzcGGTHAUB0IY16DH3k+yT\nrvvAb2woLLreXDgC4mdrUeXFEnV9GzWy33DcbY8mZo9CSYbfYwlILa56EW3UfjoB\nYR4G4sNYEuz/BlIS3UaMzAM/a8S/TTQmv1rqck0UAXhCpVK3RBD5nwkGJQEdF159\n3Fe0Cpzoj0S11oak97OvGEhzcrn0Cib5A8IQt763LL0cIXdBF5U/2FOrYz6Dobao\nYTO93TLirNFuefFe/xl2sBDUIAMqVIZWviKl0AYDIQKBgQDkZcsB3BacZNHeTkNF\na8txFIAOJfXf/jsriVMx3s7rmrSsC1J7MJwdAau3EU4UpwBXvmRh7PRHNi4M+d+B\nVCltVlbMytcP9EQbG4wznvo118vXUduYR2dHnx99eLxEP1r6FPd7Ng94vXOHgJGf\nyzR8AE6f3ZV71BQuKmemo2enAwKBgQC3iwDiIgHi0rCSMTTcRwg1inFu3/lgdd+0\njl9Jipm4+HgFTTLmNNn8lLtMgrlEaw402cErmAKnDAexjAY1cddDmPUSSF+z/juw\nJEMP7+/b9yK51W0o3W/CaI/VPKKDodfcmLaWmf2l665lhMPdAdL8pKI4ey66NguD\nmBEjgUQzzQKBgQDGY10IyoGhazmYneZ1AIMAQExS/coqvIvs7yWipeSaZhEVd+g3\nNSwFThQ3ugJWkhEMJkbPgKFW0csCXnEzx0UMwfg3fuF77DLUUQNhO3AfIMjgySuz\nhCJD5gXWWO8/YMDHhTrKfVTHDdiyFI1pfXqjUyj4V+KmYrr8try91CYOvQKBgGsT\nS8OgpUsHMP1IeZccVVyu1Hgjb99nRMeX210S35U6VwJ4w/3qzVT+C7qZ9K1GazZH\n5tqgxsNfUEmy75hUPonv/ODamgJuibG7pefxSRQhO///07voWx+mUmrLAeAbFcxa\nxwJa+c/HC6mr53vSaZz8Cqs6qcr2ZocxuqXk7r3tAoGAD4NVneZfYLA/TEeG55oe\nHB+XOWAINyRqSnrZwHSXLyed9ARoUZIUv33eXe+1X8cbXiN4174ZpJYzYU+B5jNs\nd247aGQH5Ek9fZZ9QD+NWBE3s6WknEtgQwhK9q0XyYAcYMY37QBtEf32YqRrmesy\nQ1MwtHCuZQxed725uF3QZV4=\n-----END PRIVATE KEY-----",
	//	Application: "https://self-service.qa.np.westeurope.azure.axual.cloud/api/applications/8dfd00d6281d406bb797e193c2a066b3",
	//	Environment: "https://self-service.qa.np.westeurope.azure.axual.cloud/api/environments/3d2eb1e9c97f4cf08e310cec88de2449",
	//}
	//
	//connectorApplicationPrincipal, _ := c.CreateApplicationPrincipal(array)
	//log.Printf("CREATE CONNECTOR APPLICATION: %s\n", connectorApplicationPrincipal)
	/*
		Create Connector Application
	*/
	//var connectorApplicationCreateRequest = webclient.ApplicationRequest{
	//	ApplicationType:  "Connector",
	//	ApplicationId:    "kaspar_test33",
	//	Name:             "kaspar_test33",
	//	ShortName:        "kaspar_test33",
	//	Owners:           "https://self-service.qa.np.westeurope.azure.axual.cloud/api/groups/952497db42754bfc97cef4109f59bd2a",
	//	Type:             "SINK",
	//	ApplicationClass: "com.couchbase.connect.kafka.CouchbaseSinkConnector",
	//	Visibility:       "Public",
	//	Description:      "",
	//}
	//
	//connectorApplication, _ := c.CreateApplication(connectorApplicationCreateRequest)
	//log.Printf("CREATE CONNECTOR APPLICATION: %s\n", connectorApplication)

	/*
		Get ApplicationDeployment
	*/
	//applicationDeployment, err := c.GetApplicationDeployment("8d5c7e4c7c72462abf97da8a4b4729c9")
	//if err != nil {
	//	return
	//}
	////log.Println("APPLICATION DEPLOYMENT IS", applicationDeployment.Configs)
	//
	//log.Println("APPLICATION DEPLOYMENT CONFIGS:")
	//for _, config := range applicationDeployment.Configs {
	//	log.Printf("%s: %s\n", config.ConfigKey, config.ConfigValue)
	//}

	/*
		Find ApplicationDeployment by application and environment
	*/
	//applicationDeploymentFindByApplicationAndEnvironmentResponse, err :=
	//	c.FindApplicationDeploymentByApplicationAndEnvironment("https://self-service.qa.np.westeurope.azure.axual.cloud/api/applications/6388cc48ef2942e88a340f457616e239",
	//		"https://self-service.qa.np.westeurope.azure.axual.cloud/api/environments/3d2eb1e9c97f4cf08e310cec88de2449")
	//if err != nil {
	//	return
	//}
	//log.Println("APPLICATION DEPLOYMENT IS", applicationDeployment.Configs)

	//log.Printf("FOUND APPLICATION DEPLOYMENT BY APPLICATION AND ENVIRONMENT: %s\n", applicationDeploymentFindByApplicationAndEnvironmentResponse.Embedded.ApplicationDeploymentResponses[0])

	///*
	//	Delete ApplicationDeployment
	//*/
	//applicationDeployment := c.DeleteApplicationDeployment("8d5c7e4c7c72462abf97da8a4b4729c9")
	//log.Printf("DELETE APPLICATION DEPLOYMENT: %s\n", applicationDeployment)

	/*
		Create ApplicationDeployment
	*/
	//var applicationDeploymentCreateRequest = webclient.ApplicationDeploymentCreateRequest{
	//	Application: "https://self-service.qa.np.westeurope.azure.axual.cloud/api/applications/6388cc48ef2942e88a340f457616e239",
	//	Environment: "https://self-service.qa.np.westeurope.azure.axual.cloud/api/environments/3d2eb1e9c97f4cf08e310cec88de2449",
	//	Configs: map[string]string{
	//		"config.action.reload":     "restart",
	//		"header.converter":         "",
	//		"key.converter":            "",
	//		"tasks.max":                "1",
	//		"topics":                   "1",
	//		"topics.regex":             "",
	//		"value.converter":          "",
	//		"auto.create":              "false",
	//		"auto.evolve":              "false",
	//		"fields.whitelist":         "",
	//		"pk.fields":                "",
	//		"pk.mode":                  "none",
	//		"table.name.format":        "${topic}",
	//		"table.name.normalize":     "false",
	//		"topics.to.tables.mapping": "",
	//		"connection.password":      "",
	//		"connection.url":           "2",
	//		"connection.user":          "",
	//		"db.timezone":              "UTC",
	//		"dialect.name":             "",
	//		"sql.quote.identifiers":    "true",
	//		"errors.deadletterqueue.context.headers.enable":   "false",
	//		"errors.deadletterqueue.topic.name":               "",
	//		"errors.deadletterqueue.topic.replication.factor": "3",
	//		"errors.log.enable":                               "false",
	//		"errors.log.include.messages":                     "false",
	//		"errors.retry.delay.max.ms":                       "60000",
	//		"errors.retry.timeout":                            "0",
	//		"errors.tolerance":                                "none",
	//		"predicates":                                      "",
	//		"max.retries":                                     "10",
	//		"retry.backoff.ms":                                "3000",
	//		"transforms":                                      "",
	//		"batch.size":                                      "3000",
	//		"insert.mode":                                     "insert",
	//	},
	//}
	//
	//applicationDeployment, _ := c.CreateApplicationDeployment(applicationDeploymentCreateRequest)
	//log.Printf("CREATE APPLICATION DEPLOYMENT: %s\n", applicationDeployment)

	/*
		Update ApplicationDeployment
	*/
	//var applicationDeploymentUpdateRequest = webclient.ApplicationDeploymentUpdateRequest{
	//	Configs: map[string]string{
	//		"config.action.reload":                            "restart",
	//		"topics":                                          "3",
	//		"value.converter":                                 "",
	//		"key.converter":                                   "",
	//		"header.converter":                                "",
	//		"topics.regex":                                    "",
	//		"tasks.max":                                       "1",
	//		"couchbase.bucket":                                "2",
	//		"couchbase.network":                               "test@test.com",
	//		"couchbase.seed.nodes":                            "axual",
	//		"couchbase.password":                              "qq",
	//		"couchbase.bootstrap.timeout":                     "30s",
	//		"couchbase.username":                              "q",
	//		"couchbase.durability":                            "NONE",
	//		"couchbase.replicate.to":                          "NONE",
	//		"couchbase.persist.to":                            "NONE",
	//		"errors.retry.delay.max.ms":                       "60000",
	//		"errors.log.include.messages":                     "false",
	//		"errors.log.enable":                               "false",
	//		"errors.deadletterqueue.context.headers.enable":   "false",
	//		"errors.deadletterqueue.topic.replication.factor": "3",
	//		"errors.retry.timeout":                            "0",
	//		"errors.deadletterqueue.topic.name":               "",
	//		"errors.tolerance":                                "none",
	//		"couchbase.log.redaction":                         "NONE",
	//		"couchbase.log.document.lifecycle":                "false",
	//		"couchbase.n1ql.where.fields":                     "",
	//		"couchbase.n1ql.operation":                        "UPDATE",
	//		"couchbase.n1ql.create.document":                  "true",
	//		"predicates":                                      "",
	//		"couchbase.trust.store.path":                      "",
	//		"couchbase.client.certificate.path":               "",
	//		"couchbase.trust.store.password":                  "[hidden]",
	//		"couchbase.enable.hostname.verification":          "true",
	//		"couchbase.trust.certificate.path":                "",
	//		"couchbase.enable.tls":                            "false",
	//		"couchbase.client.certificate.password":           "[hidden]",
	//		"couchbase.document.id":                           "",
	//		"couchbase.topic.to.collection":                   "",
	//		"couchbase.document.expiration":                   "0",
	//		"couchbase.sink.handler":                          "com.couchbase.connect.kafka.handler.sink.UpsertSinkHandler",
	//		"couchbase.default.collection":                    "_default._default",
	//		"couchbase.remove.document.id":                    "false",
	//		"couchbase.retry.timeout":                         "0",
	//		"couchbase.document.mode":                         "DOCUMENT",
	//		"couchbase.subdocument.path":                      "",
	//		"couchbase.subdocument.operation":                 "UPSERT",
	//		"couchbase.subdocument.create.document":           "true",
	//		"couchbase.subdocument.create.path":               "true",
	//		"transforms":                                      "",
	//	},
	//}
	//
	//applicationDeployment, _ := c.UpdateApplicationDeployment("87275f73510f44e1a95862bfdae787ca", applicationDeploymentUpdateRequest)
	//log.Printf("UPDATE APPLICATION DEPLOYMENT: %s\n", applicationDeployment)

	/*
		Create Custom Application Principal
	*/
	//var array [1]webclient.ApplicationPrincipalRequest
	//array[0] = webclient.ApplicationPrincipalRequest{
	//	Principal:   "axual-test-222222",
	//	Application: "https://platform.local/api/applications/b21cf1d63a55436391463cee3f56e393",
	//	Environment: "https://platform.local/api/7237a4093d7948228d431a603c31c904",
	//	Custom:      true,
	//}
	//applicationPrincipal, err := c.CreateApplicationPrincipal(array)
	//if err != nil {
	//	return
	//}
	//log.Println(fmt.Sprintf("applicationPrincipal %s", applicationPrincipal))

	//b00c2e07d0d34a3b81047a11bd3d3615
	//7.08
	//15:08

	/*
		Update Custom Application Principal
	*/
	//m := webclient.ApplicationPrincipalUpdateRequest{
	//	Principal: "axual-gowebclient-0000",
	//}
	//applicationPrincipal, err := c.UpdateApplicationPrincipal("b00c2e07d0d34a3b81047a11bd3d3615", m)
	//if err != nil {
	//	return
	//}
	//log.Println(fmt.Sprintf("application principal %s", applicationPrincipal))

	/*
		Get Custom Application Principal
	*/
	//applicationPrincipal, err := c.ReadApplicationPrincipal("18ceb4b241ea479392d59fc61e113132")
	//if err != nil {
	//	return
	//}
	//log.Println(fmt.Sprintf("application principal %s", applicationPrincipal))

	/*
		Delete Custom Application Principal
	*/

	//err := c.DeleteApplicationPrincipal("18ceb4b241ea479392d59fc61e113132")
	//if err != nil {
	//	return
	//}

	/*
		Delete Application Principal
	*/

	//err := c.DeleteApplicationPrincipal("6e72be22ec78497eb7603678f38ae771")
	//if err != nil {
	//	return
	//}

	/*
		Get Application Principal
	*/
	//applicationPrincipal, err := c.ReadApplicationPrincipal("6e72be22ec78497eb7603678f38ae771")
	//if err != nil {
	//	return
	//}
	//log.Println(fmt.Sprintf("application principal %s", applicationPrincipal))

	/*
		Update Application Principal
	*/
	//m := webclient.ApplicationPrincipalUpdateRequest{
	//	//10 year one
	//	Principal: "-----BEGIN CERTIFICATE-----\nMIIFuzCCA6OgAwIBAgIQKGFt4QESs5zvCkO3s1RafzANBgkqhkiG9w0BAQsFADAr\nMSkwJwYDVQQDDCBBeHVhbCBEdW1teSBJbnRlcm1lZGlhdGUgMjAxOCAwMTAeFw0y\nMTA4MDQwNzQyMzhaFw0yMjA4MTQwNzQyMzhaMGExCzAJBgNVBAYTAk5MMQswCQYD\nVQQIDAJOTDELMAkGA1UEBwwCTkwxDTALBgNVBAoMBFRlc3QxDTALBgNVBAsMBFRl\nc3QxGjAYBgNVBAMMEVZhbGlkIGluIDEwIHllYXJzMIICIjANBgkqhkiG9w0BAQEF\nAAOCAg8AMIICCgKCAgEAxj6MojhVubgxZ/6vuYHjSNlAAe0muTbb8APBhdqvDz1n\n5BMy17Y/oXpw1gOjOILGmMR73mAcdaG7HJU6lnOie6BeFsTQr86HMSOmY39BtLhh\nUyYK5EM7EXgaYLwX8vplIoO8+Jlp69k9nsv23FkLu47OqNto0/R+nlnRX61vBqY/\n/OlJCQmKJ6ZDodA9fW0oHYqyyo8fpFrAfxfHxLppnHH3S0U8+mSeuv66zW/h4+yx\nPqmKGmLBhnopAWOBE3VhHFe+2u5l159JPuATGvSxPrnJF6MZe3Cp86lFKMO+6C7O\nqiYCHGD8SmuxKLAgysZCTA4BCUE+cycxF3gHPGjzx5ZW8s9F7BKKW7Po7wWuTDIW\nQAGYDqj8skDoh+569oOfyBwJ1sQdYSpdNxmpPK9bF+QsVWmpp9vTsqhVXwK+ZEia\nmFUSSqv6C2HCcDF7ctgjUrJRncjJchVdndrg/vUnzArYgNqSAywWDHU2FanF5L9L\nSqHYvbNQIZWEZGky19tAz67g3eq7K4Y8Nug7j8JX30TlFvwZm5X8Mv07/caIi9Xp\ntpeHMSGoAjSRYbGRsKSRwIN70TZ+sJOYBIvt13kUGfRQX95Q672AlAFEfu9G3+N3\ni+fe1ObwqZ9tQ11yJJUop4+cL8iq+wH+CFaSh8Q1yQ1JTOEjMQ9YXLLBvz8CcA8C\nAwEAAaOBpDCBoTAJBgNVHRMEAjAAMB0GA1UdDgQWBBTD0BVwJqFcC4ujhwkQ0ThR\nREc1vDBJBgNVHSMEQjBAgBRr0ilXi1I2LAUKxle5GnR4rBMfZqEkpCIwIDEeMBwG\nA1UEAwwVQXh1YWwgRHVtbXkgUm9vdCAyMDE4ggIQADALBgNVHQ8EBAMCBLAwHQYD\nVR0lBBYwFAYIKwYBBQUHAwIGCCsGAQUFBwMBMA0GCSqGSIb3DQEBCwUAA4ICAQBq\nxhRmQp9QY9K5tk/NP2Lot8gRD2joNEgJ6ane+cGL7UsS4YW8vwmVDezeAMm2KH1B\nVnpIzjuGCf0vuIpBcJqA/MEpbkXiDY2JsdtOH5dc1Yn3WsYM6CZ2j7GhU6RLJcbk\ncIc6zfirr7X/BDK+hM3drityh0jMrPjew8nk905A1GTSMnQuAYZ5Nx+dZzMsrloX\n3Q7r4JtHuPPpJKGhm0Vj+giSLC4tmRk+Kp8mqONaexxWDd+q08+eNCwDo7ztEW5U\n4WneSbbbdptkzIkpWEkO044hvEyODKPLv9FOHApkYjg3njQL0rzxS2mcP2mctIxf\n8nWW1rB1C7uRHcoPgMG7J+wYQGlagOn4hEo3x3OXQLb0pgRpGmjFWPEyVAufUtgY\n8giV934Gmne+jSaME35RH9tCADai3dEVdjdHArv6x4DRa9S77rQbKsdm14P9GnQI\nO++y5ilNzPx8fI5qFMiVTgz0ZacZVbOCPUhz7hMyxsRJCCarW8MtB7efgnjogo3X\nRa+rMAnGp2z/yqAITFnzg54xrtoHitaEuBbkJ1iQW6oEv3mAIuZZVHkQY5HveknW\nTTa44LP1SvX0r6r3hpBESocbcOhYaAXoRc7VOmOZpjUzE8CxlGg7lBm7+nNVOpoX\n7QqaabYMTNRIHVxd7xwzLMJ6X4guLmCCUQsfm+Q8NQ==\n-----END CERTIFICATE-----\n-----BEGIN CERTIFICATE-----\nMIIFLTCCAxWgAwIBAgICEAAwDQYJKoZIhvcNAQELBQAwIDEeMBwGA1UEAwwVQXh1\nYWwgRHVtbXkgUm9vdCAyMDE4MB4XDTE4MDUyOTExMDEzNFoXDTI4MDUyNjExMDEz\nNFowKzEpMCcGA1UEAwwgQXh1YWwgRHVtbXkgSW50ZXJtZWRpYXRlIDIwMTggMDEw\nggIiMA0GCSqGSIb3DQEBAQUAA4ICDwAwggIKAoICAQC9uOuzJekmeo3hl8fjQlKS\nHApS3llcliq1YrXpkMbHAA9StHaMHPW+Dzr2/+cdfBAmN3sujCY8Paq15QI+TDOq\nKA5SByCBQKXx2qulBPcZs3mDMt+KxAaeWfwR4Nj0NNKbmw2HjDddo77joeVOuOX2\n4o1wXzmAAolVMIcRYA11EMWNUtYrHCzBa7RfYht2G5dE69ckrgfw1Nxs01Sbg+xP\nsK9aK/LHPUalYZNY+76x7vabEpzaPfpyKzDTWA20SPk0WfTf9/+K3o+urzDG8O/q\nw9xbBOzWohGmRyA/z841p1SD7inpZcyO/KeW1yTP2WyFxADwUrv2mEYXnma/Gdna\nG62IQYk/UMex9W8pT6tfwrg/36sSwr88yPR5dJxzjHUE+w/rYG3k+K+EqvZ5qOC5\n32AJ9BS2nbNuGpmRU1qoMCwpL7B2E/CKJLIdFcf/qmcnWJEXo+u34+fQZg8XaDCI\nXhUqAHz6YkjCiFGd/JwL1IqsfxFsV9wHTUbW2AumglU65ZrjhXrrzE7Hk9ng1spJ\ndOwfBihBNjnr0mKHY9leJ3chJ9HQ55/fEgcRNrj8EC69QCeAtpY5yOAjKpA03UvF\ngrDt8CIyIehNUwTXIhQSHZU4eZ0rzWf0vvMbhL2FvKtphbpnNKoXeNLv2IMZpT4B\nVwsqLqaIkl/I4FPpYBoSYwIDAQABo2YwZDAdBgNVHQ4EFgQUa9IpV4tSNiwFCsZX\nuRp0eKwTH2YwHwYDVR0jBBgwFoAUdKOPDqSFQ6Bfk0I/asBkByt5gsUwEgYDVR0T\nAQH/BAgwBgEB/wIBADAOBgNVHQ8BAf8EBAMCAYYwDQYJKoZIhvcNAQELBQADggIB\nAKoNIqiOdjlUBNg7cvR9Su4KgGrsZf78pG1H2MlNxJjFM/80EiWEfze/EG2MLxFq\n8vToIHDjb0kVetYpdmfHNXTTlaaroBlXwyUYToPzQ985qr3LD8RhYZFAsiZCTtpJ\n4FT6sh/mccTyx8G8+ZS6mn/le2WPj/t6beNLgbdl5n8fghdQcmT/TqGXE50UftWt\nHSx3fsq2aKuNdVzhKzTin50IbiE9DV1dKo6B+ipOy/Dz5GMv3Z/3ntLTvxabCMOl\n7s7WsUE7VPABRSifUS80Z9Ai38faLSu+Ouzx40ceXwvlFQtJ2LYQ8Ru5Q63k2wB3\nEOE6cgAhiYExrz3fDDtUkui9vIfWfTPMnXR7xQ8YqK4Qqld2ESxvMQU2jzbZKSf+\n3sWnPvN4HTg0cfysmOdLGZwf3u8A9tMtxhUEtxUx7r76M4ekSKdNv1Nf5u5N/h7b\nAbEqSp1XADTxkE448i7hNJzn2Ce6JtFya231Ni0xyYKQIajP18jNypAw1eABYFkN\n53vQTUfqcbtcrCios1xRdDqfgkYaKZv7p63aoObFTf/mmG7sFjGAEPQscagOukwN\nwnkjCVifVbk5qJUaUWSLeYziI+HYkEA9P/h4o83nbf0YgBtOFoc0XWKmKagHifZN\nSEJ9kRCWzYaL2ChiL6jHGh26WT/hbNKeAlcxPnT4u/l1\n-----END CERTIFICATE-----",
	//	//UTRECHT one
	//	//Principal: "-----BEGIN CERTIFICATE-----\nMIIF+jCCA+KgAwIBAgIRAMM8e1hKSNHPLY2DjomTTHwwDQYJKoZIhvcNAQELBQAw\nKzEpMCcGA1UEAwwgQXh1YWwgRHVtbXkgSW50ZXJtZWRpYXRlIDIwMTggMDEwHhcN\nMjExMDEyMTMwMzEzWhcNMjYxMjI1MTMwMzEzWjCBnjELMAkGA1UEBhMCTkwxEDAO\nBgNVBAgMB1V0cmVjaHQxEDAOBgNVBAcMB1V0cmVjaHQxEzARBgNVBAoMCkF4dWFs\nIEIuVi4xEzARBgNVBAsMClR1cmJ1bGVuY2UxFjAUBgNVBAMMDVRlc3RpbmcgRW1h\naWwxKTAnBgkqhkiG9w0BCQEWGnRlc3RAdHVyYnVibGVuY2UuYXh1YWwuY29tMIIC\nIjANBgkqhkiG9w0BAQEFAAOCAg8AMIICCgKCAgEAk9geBM56zC9vex1LsejgwTDL\nWs4CsUwVy3DSl2zVsB1SLti/Nekre+xpcfddgD9T6Ad4uWsajx4k6kNnWVCC5FtQ\nK/IOWsPsoJk81jvKBB7h32xraW0XxFouYk6CEMwzmBM6j9doLiy3rO1K8POFDgxQ\n19BeKVxt6W4NhAKJ6DrqKaIlFz5vO6bUMqcaAfaIlyWkUQ0TVh0Vc0MjUIqu7eE9\nc6tDb7IPMcut0oT66PHnMWcXAzrXUWcdsdq1tbzf+9g3UJ+TuMYfcm+a+p8SfCpy\nAs+9shWcmlp2UIO91bCO3itkS8SdnmW1rT0CKcBQAfUYukxCrjvkmdu4xzk/rwk1\ngOyjeC68ru3Qt4EjXaG3xhm7wqFbO5Wf9B8mZAvADDx/OyAiIH7UxmgxeZ1LTTlS\nutkr5kmxKxcBjql853pOTCo3wAXxS95lupcrC6Q4nTJbMWFVVCstWsxY/xCnIdJ6\n4RieTV81Ot3UadkMB9/KW9p8NAgUJUgEVNTwQwuAEhWbs4fF/JbWkDmM24zHhqdA\nD6UReX4iBGPBLV4DeN4zzqX1B/1LhnWAlSN0Fxxh/oEH3eSJgeUzIGJBJDAgTcDA\nuYTBqPTg2+FCpfOKZuM8Tl0DmVwTRA4RIwOPKjD3PLPos3LVhR60kNi7R1DA9pb3\nyPzzZWaCOV8C6m9HLpsCAwEAAaOBpDCBoTAJBgNVHRMEAjAAMB0GA1UdDgQWBBSW\n+L1vlkjq/YHV6t0dDjhXaI4LnTBJBgNVHSMEQjBAgBRr0ilXi1I2LAUKxle5GnR4\nrBMfZqEkpCIwIDEeMBwGA1UEAwwVQXh1YWwgRHVtbXkgUm9vdCAyMDE4ggIQADAL\nBgNVHQ8EBAMCBLAwHQYDVR0lBBYwFAYIKwYBBQUHAwIGCCsGAQUFBwMBMA0GCSqG\nSIb3DQEBCwUAA4ICAQArQ5LJ2p4XBNdHn2X3s+U2iiq1+0a/sr5v50BhrBurV9mz\nq9R0aK7pYwq5Ol+WRKRj3RYNcOsiwfeyY1GV+3BLmgctrMb4pzHvumfy0qTDnnrE\nC0UQLIOyK10uJBqwEJMt2hwLZWAaRqwfMMTXRg61i51PIEJN7OU9jeTeEVNDrrBi\nhWWIIP2RrVm2MKA85z896J+DFQ0L/aq6SJk/vUJSUsb0A49gvxYWJzkbNG5vb3OA\nxabPoDTb9EI9Q8DnWLiM/ay5Kol6niDubJ4KVgWJSLI+5KXaMFmI6zbiapcp6pc7\nnDbDmOdHiWhdYu30HSMNFnuc0GsR49NQjTW2nC7FAymjHf2mTkyRtUkXFbBbie/1\n1uzVur+XncOWydHPxHn5fCXExCQYhgWigY5Kj3Fy25vwKfCbB2Quu9669Nka3iYD\nZ5JlbaoOJ9ho2IHDiyporiacDXnH+gfSKw2cKHrthXsHRpywt48cj6FtrD8pIVBz\ngIh0c0RINucjQJR+JvH8OpbILSv2ArgyhQomCjoiGPs/NIrtttBa4sIGJylkxLMp\nmb1KE1zpRGjA9o3Lj50dids8eQ5FH7Ldo7tpfMP4bkNh983Hwr4UUXDfNMxdOfkU\n9+KEor3JlVRl8aGIBpnu34DbABbm+g1jm9xG3syH2QT7wo3ciEo0WtYg4IUJsw==\n-----END CERTIFICATE-----\n-----BEGIN CERTIFICATE-----\nMIIFLTCCAxWgAwIBAgICEAAwDQYJKoZIhvcNAQELBQAwIDEeMBwGA1UEAwwVQXh1\nYWwgRHVtbXkgUm9vdCAyMDE4MB4XDTE4MDUyOTExMDEzNFoXDTI4MDUyNjExMDEz\nNFowKzEpMCcGA1UEAwwgQXh1YWwgRHVtbXkgSW50ZXJtZWRpYXRlIDIwMTggMDEw\nggIiMA0GCSqGSIb3DQEBAQUAA4ICDwAwggIKAoICAQC9uOuzJekmeo3hl8fjQlKS\nHApS3llcliq1YrXpkMbHAA9StHaMHPW+Dzr2/+cdfBAmN3sujCY8Paq15QI+TDOq\nKA5SByCBQKXx2qulBPcZs3mDMt+KxAaeWfwR4Nj0NNKbmw2HjDddo77joeVOuOX2\n4o1wXzmAAolVMIcRYA11EMWNUtYrHCzBa7RfYht2G5dE69ckrgfw1Nxs01Sbg+xP\nsK9aK/LHPUalYZNY+76x7vabEpzaPfpyKzDTWA20SPk0WfTf9/+K3o+urzDG8O/q\nw9xbBOzWohGmRyA/z841p1SD7inpZcyO/KeW1yTP2WyFxADwUrv2mEYXnma/Gdna\nG62IQYk/UMex9W8pT6tfwrg/36sSwr88yPR5dJxzjHUE+w/rYG3k+K+EqvZ5qOC5\n32AJ9BS2nbNuGpmRU1qoMCwpL7B2E/CKJLIdFcf/qmcnWJEXo+u34+fQZg8XaDCI\nXhUqAHz6YkjCiFGd/JwL1IqsfxFsV9wHTUbW2AumglU65ZrjhXrrzE7Hk9ng1spJ\ndOwfBihBNjnr0mKHY9leJ3chJ9HQ55/fEgcRNrj8EC69QCeAtpY5yOAjKpA03UvF\ngrDt8CIyIehNUwTXIhQSHZU4eZ0rzWf0vvMbhL2FvKtphbpnNKoXeNLv2IMZpT4B\nVwsqLqaIkl/I4FPpYBoSYwIDAQABo2YwZDAdBgNVHQ4EFgQUa9IpV4tSNiwFCsZX\nuRp0eKwTH2YwHwYDVR0jBBgwFoAUdKOPDqSFQ6Bfk0I/asBkByt5gsUwEgYDVR0T\nAQH/BAgwBgEB/wIBADAOBgNVHQ8BAf8EBAMCAYYwDQYJKoZIhvcNAQELBQADggIB\nAKoNIqiOdjlUBNg7cvR9Su4KgGrsZf78pG1H2MlNxJjFM/80EiWEfze/EG2MLxFq\n8vToIHDjb0kVetYpdmfHNXTTlaaroBlXwyUYToPzQ985qr3LD8RhYZFAsiZCTtpJ\n4FT6sh/mccTyx8G8+ZS6mn/le2WPj/t6beNLgbdl5n8fghdQcmT/TqGXE50UftWt\nHSx3fsq2aKuNdVzhKzTin50IbiE9DV1dKo6B+ipOy/Dz5GMv3Z/3ntLTvxabCMOl\n7s7WsUE7VPABRSifUS80Z9Ai38faLSu+Ouzx40ceXwvlFQtJ2LYQ8Ru5Q63k2wB3\nEOE6cgAhiYExrz3fDDtUkui9vIfWfTPMnXR7xQ8YqK4Qqld2ESxvMQU2jzbZKSf+\n3sWnPvN4HTg0cfysmOdLGZwf3u8A9tMtxhUEtxUx7r76M4ekSKdNv1Nf5u5N/h7b\nAbEqSp1XADTxkE448i7hNJzn2Ce6JtFya231Ni0xyYKQIajP18jNypAw1eABYFkN\n53vQTUfqcbtcrCios1xRdDqfgkYaKZv7p63aoObFTf/mmG7sFjGAEPQscagOukwN\nwnkjCVifVbk5qJUaUWSLeYziI+HYkEA9P/h4o83nbf0YgBtOFoc0XWKmKagHifZN\nSEJ9kRCWzYaL2ChiL6jHGh26WT/hbNKeAlcxPnT4u/l1\n-----END CERTIFICATE-----\n-----BEGIN CERTIFICATE-----\nMIIFJjCCAw6gAwIBAgIJAINuAirfnRU6MA0GCSqGSIb3DQEBCwUAMCAxHjAcBgNV\nBAMMFUF4dWFsIER1bW15IFJvb3QgMjAxODAeFw0xODA1MjkxMDM0MTRaFw0zODA1\nMjQxMDM0MTRaMCAxHjAcBgNVBAMMFUF4dWFsIER1bW15IFJvb3QgMjAxODCCAiIw\nDQYJKoZIhvcNAQEBBQADggIPADCCAgoCggIBAMVDjbhq3TGuQ6INTZ+dhSIgsdbq\nw2nxF3myrS7v89bcNxMyLypWYTmR4OAYRXRBnW4KX6sTubPyL3ogPz6hXmfmPfAz\n+X//HTIiybL3e3qwxqWphp09+JT6veEp/e/wEEjSMj5nsxkDEjj9JEQWu/1B+N+V\nXOJkTYFy05ZgeWplkyLwT71myF047aISK27a+VebBMaPpvvetScbMSwxAbk51cGV\nUC4gpwvnvsbp/CRuMV0dYzkeTmxgn860l3s8+7qUJoOrtiO0cDpv97SK9Ck9ef1k\nR6KFttzxb/u+eMFi3RUErEGwE8P3thTseXRkp5hMwcyaSQv0wfLawlwcNFGOzsBx\nfJS7QUIUpEyzRqj5Ppgaj530APxbgitLOfVLZ2DvcBcmnQns6OE+uwymuvAj8Ftj\n6AFJXH2lmswHLl5uD9kIOwmpZg4NZLP2Qv+WOT6HLgI7Kv1z0OV2H7UlWA7hwQXl\noQ6fJ2YLEhT+GM9xHKJ+DQCxvjWvtGUSb/Dk0j/R9mpSFfHvVJgE/xV+7F7Vlyw5\n/cDpF3GZOTGQ/MFy4RqRrTtjnZw2/bZZyJ+Xb743OeQhABFUdadh8cmyehDregtr\nalHxtjKxCxrT55OHCYhbCoz6nEnQURD7EPQhU5puUKalRq2ApDkveIk8uj0HQmQm\nKyRuNX7M6vCoWnpxAgMBAAGjYzBhMB0GA1UdDgQWBBR0o48OpIVDoF+TQj9qwGQH\nK3mCxTAfBgNVHSMEGDAWgBR0o48OpIVDoF+TQj9qwGQHK3mCxTAPBgNVHRMBAf8E\nBTADAQH/MA4GA1UdDwEB/wQEAwIBhjANBgkqhkiG9w0BAQsFAAOCAgEAbJanqR4P\nmr05AyAu8vlrLsleXA8VAPDiaaYStYH5cIdBBWkaIxanLFDmbyQwKkKdkHQWV9X8\n1P52q49T9RsoBsEOmwdiaCY2PEUz7Y3bFW0UeM+k65VlHlXWywRM6+O02t4TrJXH\nF6h7vPon01OwhgW9Yil/Kr+yyZK50Ic+pm4UhHmtxY932cNaRCdae5tKsjabsP7Z\nrdAksLia8mTp+HADkZJ1uODxyDh0S1WMKB5JoHYBrmtUr1NYLgRC6SinhK4r7rbi\nEWuurE605Nm//jv3Czdy8gEsMDtXLZYY0iqGnD11MAJFXyQ6PG2eq1cXcsJNRojm\n8D4ipfQ+z4bp9dDVR2DzVyTYe4yuhZuIe2phOhPc8KkBaXQRMHfVKyeEmzqEFLaM\nkfaDZkRsrMZSqh+KJoxDG3h8UqssChX+cuZdsjRhNWRqfbB20I9Upwa+XooyCU4E\nEkYyFTMchtvbYZEN/XvlPfhK5JB9eJ5rrcE8hKsP3gftchWWqCDedKugvZW/t5Vk\nlc+z4IjiJFnRDfcr4Z5V2Hpseyno3AEK7aUdJlmuPnxoImFXfQ4jUguM/wznJHl7\nXv9T0oaBVHM7Bd6PlES04Oho0KZXS6NryTsZn9GFV4qGZj5lEeOVl15AOfeIjP/I\nokA2uUH/ZuJlR/BEmqbLt5HWPRNT/GgLfPY=\n-----END CERTIFICATE-----",
	//}
	//applicationPrincipal, err := c.UpdateApplicationPrincipal("6e72be22ec78497eb7603678f38ae771", m)
	//if err != nil {
	//	return
	//}
	//log.Println(fmt.Sprintf("application principal %s", applicationPrincipal))

	/*
		Create Application Principal
	*/
	//var array [1]webclient.ApplicationPrincipalRequest
	//array[0] = webclient.ApplicationPrincipalRequest{
	//	Principal:   "-----BEGIN CERTIFICATE-----\nMIIF+jCCA+KgAwIBAgIRAMM8e1hKSNHPLY2DjomTTHwwDQYJKoZIhvcNAQELBQAw\nKzEpMCcGA1UEAwwgQXh1YWwgRHVtbXkgSW50ZXJtZWRpYXRlIDIwMTggMDEwHhcN\nMjExMDEyMTMwMzEzWhcNMjYxMjI1MTMwMzEzWjCBnjELMAkGA1UEBhMCTkwxEDAO\nBgNVBAgMB1V0cmVjaHQxEDAOBgNVBAcMB1V0cmVjaHQxEzARBgNVBAoMCkF4dWFs\nIEIuVi4xEzARBgNVBAsMClR1cmJ1bGVuY2UxFjAUBgNVBAMMDVRlc3RpbmcgRW1h\naWwxKTAnBgkqhkiG9w0BCQEWGnRlc3RAdHVyYnVibGVuY2UuYXh1YWwuY29tMIIC\nIjANBgkqhkiG9w0BAQEFAAOCAg8AMIICCgKCAgEAk9geBM56zC9vex1LsejgwTDL\nWs4CsUwVy3DSl2zVsB1SLti/Nekre+xpcfddgD9T6Ad4uWsajx4k6kNnWVCC5FtQ\nK/IOWsPsoJk81jvKBB7h32xraW0XxFouYk6CEMwzmBM6j9doLiy3rO1K8POFDgxQ\n19BeKVxt6W4NhAKJ6DrqKaIlFz5vO6bUMqcaAfaIlyWkUQ0TVh0Vc0MjUIqu7eE9\nc6tDb7IPMcut0oT66PHnMWcXAzrXUWcdsdq1tbzf+9g3UJ+TuMYfcm+a+p8SfCpy\nAs+9shWcmlp2UIO91bCO3itkS8SdnmW1rT0CKcBQAfUYukxCrjvkmdu4xzk/rwk1\ngOyjeC68ru3Qt4EjXaG3xhm7wqFbO5Wf9B8mZAvADDx/OyAiIH7UxmgxeZ1LTTlS\nutkr5kmxKxcBjql853pOTCo3wAXxS95lupcrC6Q4nTJbMWFVVCstWsxY/xCnIdJ6\n4RieTV81Ot3UadkMB9/KW9p8NAgUJUgEVNTwQwuAEhWbs4fF/JbWkDmM24zHhqdA\nD6UReX4iBGPBLV4DeN4zzqX1B/1LhnWAlSN0Fxxh/oEH3eSJgeUzIGJBJDAgTcDA\nuYTBqPTg2+FCpfOKZuM8Tl0DmVwTRA4RIwOPKjD3PLPos3LVhR60kNi7R1DA9pb3\nyPzzZWaCOV8C6m9HLpsCAwEAAaOBpDCBoTAJBgNVHRMEAjAAMB0GA1UdDgQWBBSW\n+L1vlkjq/YHV6t0dDjhXaI4LnTBJBgNVHSMEQjBAgBRr0ilXi1I2LAUKxle5GnR4\nrBMfZqEkpCIwIDEeMBwGA1UEAwwVQXh1YWwgRHVtbXkgUm9vdCAyMDE4ggIQADAL\nBgNVHQ8EBAMCBLAwHQYDVR0lBBYwFAYIKwYBBQUHAwIGCCsGAQUFBwMBMA0GCSqG\nSIb3DQEBCwUAA4ICAQArQ5LJ2p4XBNdHn2X3s+U2iiq1+0a/sr5v50BhrBurV9mz\nq9R0aK7pYwq5Ol+WRKRj3RYNcOsiwfeyY1GV+3BLmgctrMb4pzHvumfy0qTDnnrE\nC0UQLIOyK10uJBqwEJMt2hwLZWAaRqwfMMTXRg61i51PIEJN7OU9jeTeEVNDrrBi\nhWWIIP2RrVm2MKA85z896J+DFQ0L/aq6SJk/vUJSUsb0A49gvxYWJzkbNG5vb3OA\nxabPoDTb9EI9Q8DnWLiM/ay5Kol6niDubJ4KVgWJSLI+5KXaMFmI6zbiapcp6pc7\nnDbDmOdHiWhdYu30HSMNFnuc0GsR49NQjTW2nC7FAymjHf2mTkyRtUkXFbBbie/1\n1uzVur+XncOWydHPxHn5fCXExCQYhgWigY5Kj3Fy25vwKfCbB2Quu9669Nka3iYD\nZ5JlbaoOJ9ho2IHDiyporiacDXnH+gfSKw2cKHrthXsHRpywt48cj6FtrD8pIVBz\ngIh0c0RINucjQJR+JvH8OpbILSv2ArgyhQomCjoiGPs/NIrtttBa4sIGJylkxLMp\nmb1KE1zpRGjA9o3Lj50dids8eQ5FH7Ldo7tpfMP4bkNh983Hwr4UUXDfNMxdOfkU\n9+KEor3JlVRl8aGIBpnu34DbABbm+g1jm9xG3syH2QT7wo3ciEo0WtYg4IUJsw==\n-----END CERTIFICATE-----\n-----BEGIN CERTIFICATE-----\nMIIFLTCCAxWgAwIBAgICEAAwDQYJKoZIhvcNAQELBQAwIDEeMBwGA1UEAwwVQXh1\nYWwgRHVtbXkgUm9vdCAyMDE4MB4XDTE4MDUyOTExMDEzNFoXDTI4MDUyNjExMDEz\nNFowKzEpMCcGA1UEAwwgQXh1YWwgRHVtbXkgSW50ZXJtZWRpYXRlIDIwMTggMDEw\nggIiMA0GCSqGSIb3DQEBAQUAA4ICDwAwggIKAoICAQC9uOuzJekmeo3hl8fjQlKS\nHApS3llcliq1YrXpkMbHAA9StHaMHPW+Dzr2/+cdfBAmN3sujCY8Paq15QI+TDOq\nKA5SByCBQKXx2qulBPcZs3mDMt+KxAaeWfwR4Nj0NNKbmw2HjDddo77joeVOuOX2\n4o1wXzmAAolVMIcRYA11EMWNUtYrHCzBa7RfYht2G5dE69ckrgfw1Nxs01Sbg+xP\nsK9aK/LHPUalYZNY+76x7vabEpzaPfpyKzDTWA20SPk0WfTf9/+K3o+urzDG8O/q\nw9xbBOzWohGmRyA/z841p1SD7inpZcyO/KeW1yTP2WyFxADwUrv2mEYXnma/Gdna\nG62IQYk/UMex9W8pT6tfwrg/36sSwr88yPR5dJxzjHUE+w/rYG3k+K+EqvZ5qOC5\n32AJ9BS2nbNuGpmRU1qoMCwpL7B2E/CKJLIdFcf/qmcnWJEXo+u34+fQZg8XaDCI\nXhUqAHz6YkjCiFGd/JwL1IqsfxFsV9wHTUbW2AumglU65ZrjhXrrzE7Hk9ng1spJ\ndOwfBihBNjnr0mKHY9leJ3chJ9HQ55/fEgcRNrj8EC69QCeAtpY5yOAjKpA03UvF\ngrDt8CIyIehNUwTXIhQSHZU4eZ0rzWf0vvMbhL2FvKtphbpnNKoXeNLv2IMZpT4B\nVwsqLqaIkl/I4FPpYBoSYwIDAQABo2YwZDAdBgNVHQ4EFgQUa9IpV4tSNiwFCsZX\nuRp0eKwTH2YwHwYDVR0jBBgwFoAUdKOPDqSFQ6Bfk0I/asBkByt5gsUwEgYDVR0T\nAQH/BAgwBgEB/wIBADAOBgNVHQ8BAf8EBAMCAYYwDQYJKoZIhvcNAQELBQADggIB\nAKoNIqiOdjlUBNg7cvR9Su4KgGrsZf78pG1H2MlNxJjFM/80EiWEfze/EG2MLxFq\n8vToIHDjb0kVetYpdmfHNXTTlaaroBlXwyUYToPzQ985qr3LD8RhYZFAsiZCTtpJ\n4FT6sh/mccTyx8G8+ZS6mn/le2WPj/t6beNLgbdl5n8fghdQcmT/TqGXE50UftWt\nHSx3fsq2aKuNdVzhKzTin50IbiE9DV1dKo6B+ipOy/Dz5GMv3Z/3ntLTvxabCMOl\n7s7WsUE7VPABRSifUS80Z9Ai38faLSu+Ouzx40ceXwvlFQtJ2LYQ8Ru5Q63k2wB3\nEOE6cgAhiYExrz3fDDtUkui9vIfWfTPMnXR7xQ8YqK4Qqld2ESxvMQU2jzbZKSf+\n3sWnPvN4HTg0cfysmOdLGZwf3u8A9tMtxhUEtxUx7r76M4ekSKdNv1Nf5u5N/h7b\nAbEqSp1XADTxkE448i7hNJzn2Ce6JtFya231Ni0xyYKQIajP18jNypAw1eABYFkN\n53vQTUfqcbtcrCios1xRdDqfgkYaKZv7p63aoObFTf/mmG7sFjGAEPQscagOukwN\nwnkjCVifVbk5qJUaUWSLeYziI+HYkEA9P/h4o83nbf0YgBtOFoc0XWKmKagHifZN\nSEJ9kRCWzYaL2ChiL6jHGh26WT/hbNKeAlcxPnT4u/l1\n-----END CERTIFICATE-----\n-----BEGIN CERTIFICATE-----\nMIIFJjCCAw6gAwIBAgIJAINuAirfnRU6MA0GCSqGSIb3DQEBCwUAMCAxHjAcBgNV\nBAMMFUF4dWFsIER1bW15IFJvb3QgMjAxODAeFw0xODA1MjkxMDM0MTRaFw0zODA1\nMjQxMDM0MTRaMCAxHjAcBgNVBAMMFUF4dWFsIER1bW15IFJvb3QgMjAxODCCAiIw\nDQYJKoZIhvcNAQEBBQADggIPADCCAgoCggIBAMVDjbhq3TGuQ6INTZ+dhSIgsdbq\nw2nxF3myrS7v89bcNxMyLypWYTmR4OAYRXRBnW4KX6sTubPyL3ogPz6hXmfmPfAz\n+X//HTIiybL3e3qwxqWphp09+JT6veEp/e/wEEjSMj5nsxkDEjj9JEQWu/1B+N+V\nXOJkTYFy05ZgeWplkyLwT71myF047aISK27a+VebBMaPpvvetScbMSwxAbk51cGV\nUC4gpwvnvsbp/CRuMV0dYzkeTmxgn860l3s8+7qUJoOrtiO0cDpv97SK9Ck9ef1k\nR6KFttzxb/u+eMFi3RUErEGwE8P3thTseXRkp5hMwcyaSQv0wfLawlwcNFGOzsBx\nfJS7QUIUpEyzRqj5Ppgaj530APxbgitLOfVLZ2DvcBcmnQns6OE+uwymuvAj8Ftj\n6AFJXH2lmswHLl5uD9kIOwmpZg4NZLP2Qv+WOT6HLgI7Kv1z0OV2H7UlWA7hwQXl\noQ6fJ2YLEhT+GM9xHKJ+DQCxvjWvtGUSb/Dk0j/R9mpSFfHvVJgE/xV+7F7Vlyw5\n/cDpF3GZOTGQ/MFy4RqRrTtjnZw2/bZZyJ+Xb743OeQhABFUdadh8cmyehDregtr\nalHxtjKxCxrT55OHCYhbCoz6nEnQURD7EPQhU5puUKalRq2ApDkveIk8uj0HQmQm\nKyRuNX7M6vCoWnpxAgMBAAGjYzBhMB0GA1UdDgQWBBR0o48OpIVDoF+TQj9qwGQH\nK3mCxTAfBgNVHSMEGDAWgBR0o48OpIVDoF+TQj9qwGQHK3mCxTAPBgNVHRMBAf8E\nBTADAQH/MA4GA1UdDwEB/wQEAwIBhjANBgkqhkiG9w0BAQsFAAOCAgEAbJanqR4P\nmr05AyAu8vlrLsleXA8VAPDiaaYStYH5cIdBBWkaIxanLFDmbyQwKkKdkHQWV9X8\n1P52q49T9RsoBsEOmwdiaCY2PEUz7Y3bFW0UeM+k65VlHlXWywRM6+O02t4TrJXH\nF6h7vPon01OwhgW9Yil/Kr+yyZK50Ic+pm4UhHmtxY932cNaRCdae5tKsjabsP7Z\nrdAksLia8mTp+HADkZJ1uODxyDh0S1WMKB5JoHYBrmtUr1NYLgRC6SinhK4r7rbi\nEWuurE605Nm//jv3Czdy8gEsMDtXLZYY0iqGnD11MAJFXyQ6PG2eq1cXcsJNRojm\n8D4ipfQ+z4bp9dDVR2DzVyTYe4yuhZuIe2phOhPc8KkBaXQRMHfVKyeEmzqEFLaM\nkfaDZkRsrMZSqh+KJoxDG3h8UqssChX+cuZdsjRhNWRqfbB20I9Upwa+XooyCU4E\nEkYyFTMchtvbYZEN/XvlPfhK5JB9eJ5rrcE8hKsP3gftchWWqCDedKugvZW/t5Vk\nlc+z4IjiJFnRDfcr4Z5V2Hpseyno3AEK7aUdJlmuPnxoImFXfQ4jUguM/wznJHl7\nXv9T0oaBVHM7Bd6PlES04Oho0KZXS6NryTsZn9GFV4qGZj5lEeOVl15AOfeIjP/I\nokA2uUH/ZuJlR/BEmqbLt5HWPRNT/GgLfPY=\n-----END CERTIFICATE-----",
	//	Application: "https://platform.local/api/applications/b21cf1d63a55436391463cee3f56e393",
	//	Environment: "https://platform.local/api/7237a4093d7948228d431a603c31c904",
	//}
	//applicationPrincipal, err := c.CreateApplicationPrincipal(array)
	//if err != nil {
	//	return
	//}
	//log.Println(fmt.Sprintf("applicationPrincipal %s", applicationPrincipal))

	//a68bc0f8869a4916a6f815325c0d4b06
	//7.08 15:07

	/*
		Delete Topic Config
	*/

	//err := c.DeleteTopicConfig("0b3e262f9303426fa8c0a2c282bde867")
	//if err != nil {
	//	return
	//}

	/*
		Create Topic Config
	*/
	//m := webclient.TopicConfigRequest{
	//	Partitions:    1,
	//	RetentionTime: 3600001,
	//	Topic:        "https://platform.local/api/streams/295e1658752940cc96925effb402cd62",
	//	Environment:   "https://platform.local/api/environments/7237a4093d7948228d431a603c31c904",
	//}
	//topicConfig, err := c.CreateTopicConfig(m)
	//if err != nil {
	//	return
	//}
	//log.Println(fmt.Sprintf("topicConfig %s", topicConfig))

	/*
		Update Topic Config
	*/
	//m := webclient.TopicConfigRequest{RetentionTime: 3600001}
	//topic, err := c.UpdateTopicConfig("d3861b6deb884f79bf43b8ecc37ef728", m)
	//if err != nil {
	//	return
	//}
	//log.Println(fmt.Sprintf("topic %s", topic))
	/*
		Get TopicConfig
	*/
	//topic, err := c.ReadTopicConfig("d3861b6deb884f79bf43b8ecc37ef728")
	//if err != nil {
	//	return
	//}
	//log.Println(fmt.Sprintf("topic %s", topic))
	/*
		Get All Groups
	*/
	//topics, err := c.GetGroups()
	//if err != nil {
	//	return
	//}
	//for i, env := range topics.Embedded.Groups {
	//	log.Println("group no. ", i)
	//	log.Println("group data/ ", env)
	//}

	/*
		Read Topic
	*/
	//schema, err := c.ReadTopic("7b68fe584eb9414cad825f90c0c283d7")
	//if err != nil {
	//	return
	//}
	//log.Println("TOPIC IS", schema)

	///*
	//	Get Schema
	//*/
	//schema, err := c.GetSchemaVersion("88927b7fe5b54d8196e349fa031f055f")
	//if err != nil {
	//	return
	//}
	//log.Println("SCHEMA IS", schema)

	/*
		Validate Schema
	*/

	//body := "{\n  \"type\" : \"record\",\n  \"name\" : \"Application\",\n  \"namespace\" : \"io.axual.client.example.schema.mark3\",\n  \"doc\" : \"Identification of an application\",\n  \"fields\" : [ {\n    \"name\" : \"goat\",\n    \"type\" : \"string\",\n    \"doc\" : \"Random propesdfasdrty223\"\n  }, {\n    \"name\" : \"name\",\n    \"type\" : \"string\",\n    \"doc\" : \"The name of the application\"\n  }, {\n    \"name\" : \"version\",\n    \"type\" : [ \"null\", \"string\" ],\n    \"doc\" : \"(Optional) The application version\",\n    \"default\" : null\n  }, {\n    \"name\" : \"owner2\",\n    \"type\" : [ \"null\", \"string\" ],\n    \"doc\" : \"The owner of the application\",\n    \"default\" : null\n  } ]\n}"
	//test := webclient.ValidateSchemaVersionRequest{
	//	Schema: body,
	//}
	//schema, err := c.ValidateSchemaVersion(test)
	//if err != nil {
	//	return
	//}
	//log.Println("SCHEMA IS", schema)

	/*
		Get Topic
	*/
	//topic, err := c.ReadTopic("a514c764c8034d4eab4087cb2f0805c8")
	//if err != nil {
	//	return
	//}
	//log.Println(fmt.Sprintf("topic %s", topic))

	/*
		Create Topic
	*/
	//topic := createTopic(c)
	//log.Println(fmt.Sprintf("topic %s", topic))

	/*
		Update Topic
	*/
	//m := map[string]interface{}{"name": "testtopic5"}
	//topic, err := c.UpdateTopic("1bc1130b24794ffebafdea32ff33b94e", m)
	//if err != nil {
	//	return
	//}
	//log.Println(fmt.Sprintf("topic %s", topic))

	/*
		Delete Topic
	*/
	//err = c.DeleteTopic("1bc1130b24794ffebafdea32ff33b94e")
	//if err != nil {
	//	return
	//}

	/*
		Read All Environments
	*/
	//envs, err := c.ReadEnvironments()
	//if err != nil {
	//	return
	//}
	//
	//for i, env := range envs.Embedded.Environments {
	//	log.Println("env no. ", i)
	//	log.Println("env data/ ", env)
	//}

	/*
		Delete Environment
	*/
	//c.DeleteEnvironment("14c1eaa312f64e7a92dd36ffaa848e12")

	/*
		Create and Delete Environment
	*/
	//env := createEnv(c)
	//log.Println(env.Uid)
	//c.DeleteEnvironment(env.Uid)
	//parseSomeData()

	/*
		Get Groups
	*/
	//groups, err := c.GetGroups()
	//if err != nil {
	//	return
	//}
	//log.Println("groups")
	//log.Println(fmt.Sprintf("groups %s", groups))

	/*
		Testing Tokens
	*/
	//credentials := examplePasswordCredentials()
	//log.Println("refresh token")
	//log.Println(credentials.RefreshToken)
	//log.Println("access token")
	//log.Println(credentials.AccessToken)
	//log.Println("token type")
	//log.Println(credentials.TokenType)
}

//func examplePasswordCredentials() *oauth2.Token {
//	// Brightbox username and password
//	userName := "kubernetes@axual.com"
//	password := "PLEASE_CHANGE_PASSWORD"
//	// Users can have multiple accounts, so you need to specify which one
//	//accountId := "acc-h3nbk"
//	// These OAuth2 application credentials are public, distributed with the
//	// cli.
//	clientId := "self-service"
//	//applicationSecret := "mocbuipbiaa6k6c"
//
//	// Setup OAuth2 authentication.
//	conf := oauth2.Config{
//		ClientID: clientId,
//		//ClientSecret: applicationSecret,
//		Endpoint: oauth2.Endpoint{
//			TokenURL: "https://platform.local/auth/realms/axual/protocol/openid-connect/token",
//		},
//	}
//	token, err := conf.PasswordCredentialsToken(context.Background(), userName, password)
//	if err != nil {
//		fmt.Println(err)
//	}
//
//	conf.Client(context.Background(), token)
//	return token
//}

func getClient() *webclient.Client {
	apiUrl := "https://platform.local/api"
	realm := "axual"
	auth := webclient.AuthStruct{
		Username: "kubernetes@axual.com",
		Password: "PLEASE_CHANGE_PASSWORD",
		ClientId: "self-service",
		Url:      "https://platform.local/auth/realms/axual/protocol/openid-connect/token",
		Scopes:   []string{"openid", "profile", "email"},
	}

	log.SetFlags(log.LstdFlags | log.Lshortfile)

	client, err := webclient.NewClient(apiUrl, realm, auth)
	if err != nil {
		log.Println("Error:", err)
		return &webclient.Client{}
	}
	return client
}

//func createEnv(c *webclient.Client) *webclient.EnvironmentResponse {
//	request := webclient.EnvironmentRequest{
//		Name:      "test",
//		ShortName: "test",
//		//Description:         "some desc",
//		//RetentionTime:       3600000,
//		//Partitions:          3,
//		//AuthorizationIssuer: "Auto",
//		//Visibility:          "Private",
//		Instance: "https://platform.local/instances/51be2a6a5eee481198787dc346ab6608",
//		Owners:   "https://platform.local/settings/groups/dd84b3ee8e4341fbb58704b18c10ec5c",
//		//Properties:          props(),
//	}
//	environment, err := c.CreateEnvironment(request)
//	if err != nil {
//		return nil
//	}
//
//	retrieved, err := c.ReadEnvironment(environment.Uid)
//	log.Println(retrieved.Properties)
//	log.Println(retrieved)
//
//	return environment
//
//}

//func createTopic(c *webclient.Client) *webclient.TopicResponse {
//	request := webclient.TopicRequest{
//		Name:            "test11",
//		Description:     "some desc",
//		Owners:          "https://platform.local/settings/groups/dd84b3ee8e4341fbb58704b18c10ec5c",
//		KeyType:         "JSON",
//		ValueType:       "JSON",
//		RetentionPolicy: "Compact",
//		//Properties:          props(),
//	}
//	topic, err := c.CreateTopic(request)
//	if err != nil {
//		return nil
//	}
//
//	retrieved, err := c.ReadTopic(topic.Uid)
//	log.Println(retrieved.Properties)
//	log.Println(retrieved)
//
//	return topic
//
//}
