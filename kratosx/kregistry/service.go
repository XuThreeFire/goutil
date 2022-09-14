package kregistry

import (
	"encoding/json"
	"os"
	"strconv"
	"strings"

	"git.17usoft.com/go/etcdop"
	"github.com/pkg/errors"

	"github.com/go-kratos/kratos/v2/registry"
)

func marshal(r *Registry, si *registry.ServiceInstance) (string, error) {
	// get weight
	var weight int
	if r.opts.weight > 0 {
		weight = r.opts.weight
	} else if r.opts.useSystemWeight {
		weight = etcdop.GetSystemWeight()
	} else {
		weight = 10
	}

	tag := map[string]string{}
	for _, endpoint := range si.Endpoints {
		if strings.HasPrefix(endpoint, "http://") {
			tag["http"] = endpoint
		}
		if strings.HasPrefix(endpoint, "grpc://") {
			tag["grpc"] = endpoint
		}
	}
	for k, v := range si.Metadata {
		tag[k] = v
	}
	tagStr, err := json.Marshal(tag)
	if err != nil {
		return "", errors.WithMessage(err, "json.Marshal")
	}

	node := etcdop.Node{
		Key:     r.prefix + r.service,
		Name:    si.Name,
		ID:      r.id,
		Pwd:     etcdop.GetRunPath(),
		Host:    r.Host,
		Port:    r.Port,
		Pid:     os.Getpid(),
		Version: si.Version,
		Weight:  weight,
		Status:  "",
		Tag:     string(tagStr),
	}

	data, err := json.Marshal(node)
	if err != nil {
		return "", errors.WithMessage(err, "json.Marshal")
	}
	return string(data), nil
}

func unmarshal(data []byte) (si *registry.ServiceInstance, err error) {
	var node etcdop.Node
	err = json.Unmarshal(data, &node)
	if err != nil {
		return nil, err
	}

	si = &registry.ServiceInstance{}
	si.ID = strconv.FormatUint(uint64(node.ID), 10)
	si.Name = node.Name
	si.Version = node.Version
	si.Endpoints = append(si.Endpoints, "http://"+node.Host+":"+strconv.Itoa(node.Port))
	si.Metadata = make(map[string]string)
	si.Metadata["pwd"] = node.Pwd
	si.Metadata["pid"] = strconv.Itoa(node.Pid)
	si.Metadata["weight"] = strconv.Itoa(node.Weight)
	si.Metadata["status"] = node.Status

	if node.Tag != "" {
		md := make(map[string]string)
		if err := json.Unmarshal([]byte(node.Tag), &md); err == nil {
			for k, v := range md {
				si.Metadata[k] = v
			}
		}
	}

	return
}
