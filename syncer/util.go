package syncer

import (
	"context"
	"fmt"
	"github.com/loft-sh/vcluster-sdk/log"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/record"
	"reflect"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func NewGenericCreator(localClient client.Client, eventRecorder record.EventRecorder, name string) *GenericCreator {
	return &GenericCreator{
		localClient:   localClient,
		eventRecorder: eventRecorder,
		name:          name,
	}
}

type GenericCreator struct {
	localClient   client.Client
	eventRecorder record.EventRecorder

	name string
}

func (g *GenericCreator) Create(ctx context.Context, vObj, pObj client.Object, log log.Logger) (ctrl.Result, error) {
	log.Infof("create physical %s %s/%s", g.name, pObj.GetNamespace(), pObj.GetName())
	err := g.localClient.Create(ctx, pObj)
	if err != nil {
		log.Infof("error syncing %s %s/%s to physical cluster: %v", g.name, vObj.GetNamespace(), vObj.GetName(), err)
		g.eventRecorder.Eventf(vObj, "Warning", "SyncError", "Error syncing to physical cluster: %v", err)
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (g *GenericCreator) Update(ctx context.Context, vObj, pObj client.Object, log log.Logger) (ctrl.Result, error) {
	// this is needed because of interface nil check
	if !(pObj == nil || (reflect.ValueOf(pObj).Kind() == reflect.Ptr && reflect.ValueOf(pObj).IsNil())) {
		log.Infof("updating physical %s/%s, because virtual %s have changed", pObj.GetNamespace(), pObj.GetName(), g.name)
		err := g.localClient.Update(ctx, pObj)
		if err != nil {
			g.eventRecorder.Eventf(vObj, "Warning", "SyncError", "Error syncing to physical cluster: %v", err)
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

func GetByIndex(ctx context.Context, c client.Client, obj runtime.Object, index, value string) error {
	gvk, err := GVKFrom(obj, c.Scheme())
	if err != nil {
		return err
	}

	list, err := c.Scheme().New(gvk.GroupVersion().WithKind(gvk.Kind + "List"))
	if err != nil {
		// TODO: handle runtime.IsNotRegisteredError(err)
		return err
	}

	err = c.List(ctx, list.(client.ObjectList), client.MatchingFields{index: value})
	if err != nil {
		return err
	}

	objs, err := meta.ExtractList(list)
	if err != nil {
		return err
	} else if len(objs) == 0 {
		return kerrors.NewNotFound(schema.GroupResource{Group: gvk.Group}, value)
	} else if len(objs) > 1 {
		return kerrors.NewConflict(schema.GroupResource{Group: gvk.Group}, value, fmt.Errorf("more than 1 object with the value"))
	}

	val := reflect.ValueOf(obj)
	if val.Kind() != reflect.Ptr {
		return fmt.Errorf("object not a pointer")
	}

	val = val.Elem()
	newVal := reflect.Indirect(reflect.ValueOf(objs[0]))
	if !val.Type().AssignableTo(newVal.Type()) {
		return fmt.Errorf("mismatched types")
	}

	val.Set(newVal)
	return nil
}

func GVKFrom(obj runtime.Object, scheme *runtime.Scheme) (schema.GroupVersionKind, error) {
	gvks, _, err := scheme.ObjectKinds(obj)
	if err != nil {
		return schema.GroupVersionKind{}, err
	} else if len(gvks) != 1 {
		return schema.GroupVersionKind{}, fmt.Errorf("unexpected number of object kinds: %d", len(gvks))
	}

	return gvks[0], nil
}
