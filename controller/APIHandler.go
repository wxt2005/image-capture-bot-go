package controller

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/wxt2005/image-capture-bot-go/service"
)

func APIHandler(w http.ResponseWriter, r *http.Request) {
	serviceManager := service.GetServiceManager()
	header := w.Header()
	header["Content-Type"] = []string{"application/json; charset=utf-8"}
	var output Response

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(500)
		return
	}

	resp := struct {
		URLList *[]string `json:"url"`
		Force   bool      `json:"force"`
	}{
		Force: false,
	}

	err = json.Unmarshal(body, &resp)
	if err != nil {
		w.WriteHeader(500)
		return
	}

	skipCheckDuplicate := resp.Force
	var mediaList []*service.Media
	var duplicates []*service.IncomingURL
	urlStringList := resp.URLList
	incomingURLList := serviceManager.BuildIncomingURL(urlStringList)
	if skipCheckDuplicate != true {
		incomingURLList, duplicates = extractDuplicate(incomingURLList)
	}

	if len(duplicates) > 0 {
		output.Message = MsgDuplicate
		jsonByte, _ := json.Marshal(output)
		fmt.Fprintf(w, string(jsonByte))
		return
	}

	mediaList = append(mediaList, serviceManager.ExtraMediaFromURL(incomingURLList)...)

	if len(mediaList) > 0 {
		serviceManager.ConsumeMedia(mediaList)
	}

	output.Media = &mediaList
	output.Message = MsgSuccess
	jsonByte, _ := json.Marshal(output)
	fmt.Fprintf(w, string(jsonByte))
}
