package etcd

import (
	"context"
	"github.com/henryse/registrator/bridge"
	"go.etcd.io/etcd/client"
	"log"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

func init() {
	bridge.Register(new(Factory), "etcd")
}

type Factory struct{}

func (f *Factory) New(uri *url.URL) bridge.RegistryAdapter {
	urls := make([]string, 0)
	if uri.Host != "" {
		urls = append(urls, "http://"+uri.Host)
	} else {
		urls = append(urls, "http://127.0.0.1:2379")
	}

	cfg := client.Config{
		Endpoints: urls,
		Transport: client.DefaultTransport,
		// set timeout per request to fail fast when the target endpoint is unavailable
		HeaderTimeoutPerRequest: time.Second,
	}
	c, err := client.New(cfg)

	if err != nil {
		log.Fatal(err)
	}

	kapi := client.NewKeysAPI(c)

	return &Adapter{client: c, kapi: kapi, path: uri.Path, urls: urls}
}

type Adapter struct {
	client client.Client
	kapi   client.KeysAPI
	path   string
	urls   []string
}

func (r *Adapter) Ping() error {
	r.syncEtcdCluster()

	res, err := http.Get(r.urls[0] + "/version")
	// TODO: Clean up
	defer res.Body.Close()
	//body, _ := ioutil.ReadAll(res.Body)

	return err
	//rr := etcd.NewRawRequest("GET", "version", nil, nil)
	//_, err := r.client.SendRequest(rr)
	//
	//if err != nil {
	//	return err
	//}
	//return nil
}

func (r *Adapter) syncEtcdCluster() {
	//var result bool
	//result = r.client.SyncCluster()
	err := r.client.Sync(context.Background())

	if err != nil {
		log.Println(err)
	}
}

func (r *Adapter) servicePath(service *bridge.Service) string {
	return r.path + "/" + service.Name + "/" + service.ID
}

func (r *Adapter) etcdSet(service *bridge.Service, key string, value string) error {
	ttl := time.Duration(service.TTL)
	options := client.SetOptions{TTL: ttl}
	_, err := r.kapi.Set(context.Background(), key, value, &options)

	return err
}

func (r *Adapter) setValue(service *bridge.Service, key string, value string) error {
	r.syncEtcdCluster()
	path := r.servicePath(service) + "/" + key

	//	_, err := r.client.Set(path, value, uint64(service.TTL))
	err := r.etcdSet(service, path, value)
	if err != nil {
		log.Println("etcd: failed to register service:", err)
	}

	return err
}

func (r *Adapter) setTags(service *bridge.Service) error {
	r.syncEtcdCluster()
	path := r.servicePath(service) + "/tags"

	var err error
	var returnErr error
	for index, element := range service.Tags {
		//		_, err = r.client.Set(path+"/"+strconv.Itoa(index), element, uint64(service.TTL))
		err = r.etcdSet(service, path+"/"+strconv.Itoa(index), element)
		if err != nil {
			log.Println("etcd: failed to register service:", err)
			returnErr = err
		}
	}

	return returnErr
}

func (r *Adapter) setAttrs(service *bridge.Service) error {
	r.syncEtcdCluster()
	path := r.servicePath(service) + "/attrs"

	var returnErr error
	for key, value := range service.Attrs {
		//_, err := r.client.Set(path+"/"+key, value, uint64(service.TTL))
		err := r.etcdSet(service, path+"/"+key, value)

		if err != nil {
			log.Println("etcd: failed to register service:", err)
			returnErr = err
		}
	}

	return returnErr
}

func (r *Adapter) Register(service *bridge.Service) error {
	var err error
	var returnErr error

	err = r.setValue(service, "address", net.JoinHostPort(service.IP, strconv.Itoa(service.Port)))

	if err != nil {
		returnErr = err
	}

	err = r.setValue(service, "port_type", service.Origin.PortType)

	if err != nil {
		returnErr = err
	}

	err = r.setValue(service, "host_port", service.Origin.HostPort)

	if err != nil {
		returnErr = err
	}

	err = r.setValue(service, "host_ip", service.Origin.HostIP)

	if err != nil {
		returnErr = err
	}

	err = r.setValue(service, "exposed_port", service.Origin.ExposedPort)

	if err != nil {
		returnErr = err
	}

	err = r.setValue(service, "exposed_ip", service.Origin.ExposedIP)

	if err != nil {
		returnErr = err
	}

	err = r.setTags(service)

	if err != nil {
		returnErr = err
	}

	err = r.setAttrs(service)

	return returnErr
}

func (r *Adapter) Deregister(service *bridge.Service) error {
	r.syncEtcdCluster()
	return nil
}

func (r *Adapter) Refresh(service *bridge.Service) error {
	return r.Register(service)
}

func (r *Adapter) Services() ([]*bridge.Service, error) {
	return []*bridge.Service{}, nil
}
