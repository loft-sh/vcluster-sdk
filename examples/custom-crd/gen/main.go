package main

import (
	"fmt"
	"log"

	"github.com/ghodss/yaml"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-tools/pkg/crd"
	crdmarkers "sigs.k8s.io/controller-tools/pkg/crd/markers"
	"sigs.k8s.io/controller-tools/pkg/loader"
	"sigs.k8s.io/controller-tools/pkg/markers"
)

func main() {
	pkgs, err := loader.LoadRoots("./apis/v1")
	if err != nil {
		log.Fatal(err)
	}

	reg := &markers.Registry{}
	err = crdmarkers.Register(reg)
	if err != nil {
		log.Fatal(err)
	}
	parser := &crd.Parser{
		Collector: &markers.Collector{Registry: reg},
		Checker:   &loader.TypeChecker{},
	}
	outputCRD(parser, pkgs[0], "Car", "example.loft.sh", apiextensionsv1.NamespaceScoped)
}

func outputCRD(parser *crd.Parser, v1Pkg *loader.Package, kind, group string, scope apiextensionsv1.ResourceScope) {
	crd.AddKnownTypes(parser)

	parser.NeedPackage(v1Pkg)

	groupKind := schema.GroupKind{Kind: kind, Group: group}
	parser.NeedCRDFor(groupKind, nil)
	crd, ok := parser.CustomResourceDefinitions[groupKind]
	if ok {
		crd.Spec.Scope = scope
		out, err := yaml.Marshal(crd)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Println(string(out))
	} else {
		log.Fatal("Not found")
	}
}
