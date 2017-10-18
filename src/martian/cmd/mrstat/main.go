//
// Copyright (c) 2014 10X Genomics, Inc. All rights reserved.

/*
Martian status query tool

This tool is used to query or modify running instances of mrp.  Given the
path to a pipestance root directory, it attempts to discover the tcp endpoint
exposed by the mrp instance running in that directory.

The default action is to query the pipestance and return basic information
about its state.

The --stop option allows users to terminate the pipestance.  For running
pipestances, this forces the pipestance into a failed state, and mrp to
terminate.  For completed mrp instances launched with the --noexit option,
it causes mrp to terminate.

*/
package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"martian/core"
	"martian/util"
	"net/http"
	"net/url"
	"os"
	"path"
	"sort"

	"github.com/martian-lang/docopt.go"
)

func main() {
	//=========================================================================
	// Commandline argument and environment variables.
	//=========================================================================
	// Parse commandline.
	doc := `Martian Pipeline query command line tool.

Usage:
    mrstat <pipestance_name> [options]
    mrstat -h | --help | --version

Options:
    --stop      Cause the mrp process to shut down.
                If the pipestance is running, this will cause it to fail.

    -h --help   Show this message.
    --version   Show version.`
	martianVersion := util.GetVersion()
	opts, _ := docopt.Parse(doc, nil, true, martianVersion, false)

	stop := (opts["--stop"] != nil && opts["--stop"].(bool))

	psid := opts["<pipestance_name>"].(string)

	var mrpUrl *url.URL
	if urlBytes, err := ioutil.ReadFile(path.Join(psid, core.UiPort.FileName())); err != nil {
		if os.IsNotExist(err) {
			if info, err := os.Stat(psid); err != nil || !info.IsDir() {
				fmt.Fprintln(os.Stderr, psid,
					"is not a pipestance directory.")
			} else {
				fmt.Fprintln(os.Stderr, "Either ", psid,
					"is not currently running, or its monitoring UI port is disabled.")
			}
		} else {
			fmt.Fprintln(os.Stderr, "Cannot read", psid, ":", err)
		}
		os.Exit(3)
	} else if mrpUrl, err = url.Parse(string(urlBytes)); err != nil {
		fmt.Fprintln(os.Stderr, "Cannot parse url", string(urlBytes), "-", err)
		os.Exit(4)
	}
	if stop {
		sendStop(psid, mrpUrl)
	} else {
		status(psid, mrpUrl)
	}
}

func sendStop(psid string, mrpUrl *url.URL) {
	mrpUrl.Path = "api/kill"
	fmt.Println("Sending stop command to", psid)
	if resp, err := http.PostForm(mrpUrl.String(), mrpUrl.Query()); err != nil {
		fmt.Fprintln(os.Stderr, "Cannot connect to", mrpUrl, "-", err)
		os.Exit(5)
	} else {
		if resp.StatusCode != http.StatusOK {
			fmt.Fprintln(os.Stderr, "Response from mrp server:", resp.Status)
			resp.Body.Close()
			os.Exit(6)
		} else {
			fmt.Println("Stop request for", psid, "accepted")
		}
		resp.Body.Close()
	}
	os.Exit(0)
}

func status(psid string, mrpUrl *url.URL) {
	mrpUrl.Path = "/api/get-info/" + psid
	if resp, err := http.Get(mrpUrl.String()); err != nil {
		fmt.Fprintln(os.Stderr, "Cannot connect to", mrpUrl, "-", err)
		os.Exit(5)
	} else if resp.StatusCode != http.StatusOK {
		fmt.Fprintln(os.Stderr, "Response from mrp server:", resp.Status)
		resp.Body.Close()
		os.Exit(6)
	} else if bytes, err := ioutil.ReadAll(resp.Body); err != nil {
		fmt.Fprintln(os.Stderr, "Response from mrp server:", resp.Status)
		resp.Body.Close()
		os.Exit(7)
	} else {
		info := make(map[string]interface{})
		if err := json.Unmarshal(bytes, &info); err != nil {
			fmt.Fprintln(os.Stderr, "Can't parse response: ", err)
			fmt.Println(string(bytes))
		} else {
			keys := make([]string, 0, len(info))
			longest := 0
			for key := range info {
				keys = append(keys, key)
				if len(key) > longest {
					longest = len(key)
				}
			}
			sort.Strings(keys)
			for _, key := range keys {
				fmt.Printf("%*s: %v\n", longest, key, info[key])
			}
		}
		resp.Body.Close()
		os.Exit(0)
	}
}
