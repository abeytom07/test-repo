package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
)

type Api struct {
	name    string
	calls   int64
	asset   string
	service string
}

func main() {
	client := getHttpClient()
	payload := asGqlPayload(servicesGQL)
	respBytes := executeGraphQL(client, payload)
	node, err := ParseBytes(respBytes)
	if err != nil {
		panic(err)
	}
	out, err := os.OpenFile("file.csv", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0744)
	if err != nil {
		panic(err)
	}
	defer out.Close()
	results := AsMap(node).GetMap("data", "entities").GetArray("results")
	for _, node := range results.Items() {
		svc := AsMap(node)
		serviceName := svc.GetString("name")
		fmt.Printf("processing service %v\n", serviceName)
		id := svc.GetString("id")
		gql := strings.ReplaceAll(apisGql, "${SERVICE_ID}", id)
		payload = asGqlPayload(gql)
		respBytes = executeGraphQL(client, payload)
		apiRoot, err := ParseBytes(respBytes)
		if err != nil {
			panic(err)
		}
		apis := AsMap(apiRoot).GetMap("data", "entities").GetArray("results")
		assetMap := make(map[string][]Api)
		fmt.Printf("processing apis for service %v %v\n", serviceName, len(apis.Items()))
		for _, jsonNode := range apis.Items() {
			api := AsMap(jsonNode)
			apiName := api.GetString("name")
			endpointType := api.GetString("endpointType")
			callVal := api.GetValue("numCalls", "sum", "value")
			var calls int64
			if callVal != nil && callVal.Value() != nil {
				calls = int64(callVal.Val.(float64))
			}
			if strings.Contains(apiName, " ") {
				apiName = strings.Split(apiName, " ")[1]
			}
			assetId := toAssetId(serviceName, apiName, endpointType)
			apiList := assetMap[assetId]
			apiList = append(apiList, Api{
				name:    apiName,
				calls:   calls,
				asset:   assetId,
				service: serviceName,
			})
			assetMap[assetId] = apiList
		}
		printCsv(assetMap, out)
	}

}

func printCsv(assetMap map[string][]Api, out io.Writer) {
	for asset, apis := range assetMap {
		var service string
		var total int64
		for _, api := range apis {
			service = api.service
			total += api.calls
		}
		_, err := fmt.Fprintf(out, "%v,%v,%v\n", service, asset, total)
		if err != nil {
			panic(err)
		}
	}
}

var nonWordRx = regexp.MustCompile("\\W")
var multiDashRx = regexp.MustCompile("-+")

func toAssetId(serviceName string, apiUrl string, endpointType string) string {
	prefix := strings.Trim(nonWordRx.ReplaceAllString(serviceName, "_"), "_")
	groupKey := getGroupKey(apiUrl, endpointType)
	suffix := strings.Trim(nonWordRx.ReplaceAllString(groupKey, "-"), "-")
	merged := prefix + "-" + suffix
	return strings.ToLower(multiDashRx.ReplaceAllString(merged, "-"))
}

func getGroupKey(apiUrl string, endpointType string) string {
	soap := endpointType == "SOAP"
	if soap {
		index := strings.Index(apiUrl, "#")
		if index > 0 {
			apiUrl = apiUrl[:index]
		}
	}
	var segments []string
	if strings.Contains(apiUrl, "/") {
		segments = strings.Split(apiUrl, "/")
	} else if strings.Contains(apiUrl, ".") {
		segments = strings.Split(apiUrl, ".")
	} else {
		segments = []string{apiUrl}
	}
	segments = segments[1:]
	delim := "/"
	if soap {
		return delim + strings.Join(segments, delim)
	}
	var key string
	switch len(segments) {
	case 0:
		key = delim
	case 1:
		key = delim + segments[0]
	case 2:
		key = delim + segments[0]
	default:
		key = delim + strings.Join([]string{segments[0], segments[1]}, delim)
	}
	return key
}

func asGqlPayload(gql string) string {
	jsonMap := make(map[string]interface{})
	jsonMap["variables"] = make(map[string]string)
	jsonMap["query"] = gql
	marshal, err := json.Marshal(jsonMap)
	if err != nil {
		panic(err)
	}
	return string(marshal)
}

func executeGraphQL(client *http.Client, payload string) []byte {
	request, err := http.NewRequest("POST", endpoint, strings.NewReader(payload))
	if err != nil {
		panic(err)
	}
	request.Header.Add("Authorization", token)
	resp, err := client.Do(request)
	if err != nil {
		panic(err)
	}
	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	if resp.StatusCode != 200 {
		fmt.Println(resp.Status)
		fmt.Println(string(respBytes))
		panic(string(respBytes))
	}

	return respBytes
}

func getHttpClient() *http.Client {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	return &http.Client{Transport: tr}
}
