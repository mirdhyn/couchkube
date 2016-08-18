package main

import (
	"fmt"

	"k8s.io/kubernetes/pkg/api"
	vapi "k8s.io/kubernetes/pkg/api/unversioned"
	"k8s.io/kubernetes/pkg/apis/extensions"
)

// CreateDeployment takes a database name and create a K8S deployment
// of klaemo/couchdb container with label "database:<name>""
func CreateDeployment(db Database) (err error) {
	name := *db.Name

	deploySpec := &extensions.Deployment{
		TypeMeta: vapi.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "extensions/v1beta1",
		},
		ObjectMeta: api.ObjectMeta{
			Name: name,
		},
		Spec: extensions.DeploymentSpec{
			Replicas: 1,
			Template: api.PodTemplateSpec{
				ObjectMeta: api.ObjectMeta{
					Name:   name,
					Labels: map[string]string{"database": name},
				},
				Spec: api.PodSpec{
					Containers: []api.Container{
						api.Container{
							Name:  "couchdb",
							Image: "klaemo/couchdb:latest",
							Ports: []api.ContainerPort{
								api.ContainerPort{ContainerPort: 5984, Protocol: api.ProtocolTCP},
							},
							ImagePullPolicy: api.PullIfNotPresent,
							Env: []api.EnvVar{
								api.EnvVar{
									Name:  "COUCHDB_USER",
									Value: couchdbUser,
								},
								api.EnvVar{
									Name:  "COUCHDB_PASSWORD",
									Value: couchdbPassword,
								},
							},
						},
					},
				},
			},
		},
	}

	deploy := kube.Extensions().Deployments(namespace)
	_, err = deploy.Create(deploySpec)
	return err

}

// CreateService takes a database name and create a K8S service with that name
func CreateService(db Database) (err error) {
	name := *db.Name
	// Define service spec.
	serviceSpec := &api.Service{
		TypeMeta: vapi.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		ObjectMeta: api.ObjectMeta{
			Name: name,
		},
		Spec: api.ServiceSpec{
			Type:     api.ServiceTypeLoadBalancer,
			Selector: map[string]string{"database": name},
			Ports: []api.ServicePort{
				api.ServicePort{
					Protocol: api.ProtocolTCP,
					Port:     5984,
				},
			},
		},
	}

	_, err = kube.Services(namespace).Create(serviceSpec)
	if err != nil {
		return fmt.Errorf("failed to create service '%s': %s", name, err)
	}

	go CreateDB(name)

	return nil
}

// GetService takes a database name and return service its LoadBalancer endpoint
func GetService(name string) (endpoint string, err error) {
	s, err := kube.Services(namespace).Get(name)
	if err != nil {
		return "", fmt.Errorf("Database '%s' does not exist", name)
	}

	ingress := s.Status.LoadBalancer.Ingress
	if len(ingress) > 0 {
		ip := ingress[0].IP
		hostname := ingress[0].Hostname
		port := s.Spec.Ports[0].Port
		switch {
		case len(ip) != 0:
			return fmt.Sprintf("http://%s:%d", ip, port), nil
		case len(hostname) != 0:
			return fmt.Sprintf("http://%s:%d", hostname, port), nil
		default:
			return "", fmt.Errorf("failed to find service '%s'", name)
		}
	}
	return "", nil
}

// GetServicesName returns names of all services in the namespace
func GetServicesName() (names []string) {
	listOptions := api.ListOptions{
		TypeMeta: vapi.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
	}

	services, err := kube.Services(namespace).List(listOptions)
	if err != nil {
		return names
	}

	for _, s := range services.Items {
		names = append(names, s.Spec.Selector["database"])
	}
	return names
}

// CreateNamespace create namespace if not present
func CreateNamespace() error {
	namespaceSpec := &api.Namespace{
		TypeMeta: vapi.TypeMeta{
			Kind:       "Namespace",
			APIVersion: "v1",
		},
		ObjectMeta: api.ObjectMeta{
			Name: namespace,
		},
	}

	if _, err := kube.Namespaces().Get(namespace); err != nil {
		_, err = kube.Namespaces().Create(namespaceSpec)
		if err != nil {
			return err
		}
	}
	return nil
}
