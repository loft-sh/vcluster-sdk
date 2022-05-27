package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/loft-sh/vcluster-sdk/hook"
	"github.com/loft-sh/vcluster-sdk/plugin/remote"
)

type pluginServer struct {
	remote.UnimplementedPluginServer

	hooks           map[ApiVersionKindType][]hook.ClientHook
	registeredHooks []*remote.ClientHook
}

type ApiVersionKindType struct {
	ApiVersion string
	Kind       string
	Type       string
}

var _ remote.PluginServer = &pluginServer{}

func (p *pluginServer) Mutate(ctx context.Context, req *remote.MutateRequest) (*remote.MutateResult, error) {
	hooks, ok := p.hooks[ApiVersionKindType{
		ApiVersion: req.ApiVersion,
		Kind:       req.Kind,
		Type:       req.Type,
	}]
	if !ok {
		return &remote.MutateResult{}, nil
	}

	object := req.Object
	originalObject := object

	for _, h := range hooks {
		res := h.Resource()
		err := json.Unmarshal([]byte(object), res)
		if err != nil {
			return nil, fmt.Errorf("error decoding object: %v", err)
		}

		switch req.Type {
		case "CreatePhysical":
			m, ok := h.(hook.MutateCreatePhysical)
			if !ok {
				continue
			}

			res, err = m.MutateCreatePhysical(ctx, res)
			if err != nil {
				return nil, fmt.Errorf("error mutating object: %v", err)
			}
		case "UpdatePhysical":
			m, ok := h.(hook.MutateUpdatePhysical)
			if !ok {
				continue
			}

			res, err = m.MutateUpdatePhysical(ctx, res)
			if err != nil {
				return nil, fmt.Errorf("error mutating object: %v", err)
			}
		case "DeletePhysical":
			m, ok := h.(hook.MutateDeletePhysical)
			if !ok {
				continue
			}

			res, err = m.MutateDeletePhysical(ctx, res)
			if err != nil {
				return nil, fmt.Errorf("error mutating object: %v", err)
			}
		case "GetPhysical":
			m, ok := h.(hook.MutateGetPhysical)
			if !ok {
				continue
			}

			res, err = m.MutateGetPhysical(ctx, res)
			if err != nil {
				return nil, fmt.Errorf("error mutating object: %v", err)
			}
		case "CreateVirtual":
			m, ok := h.(hook.MutateCreateVirtual)
			if !ok {
				continue
			}

			res, err = m.MutateCreateVirtual(ctx, res)
			if err != nil {
				return nil, fmt.Errorf("error mutating object: %v", err)
			}
		case "UpdateVirtual":
			m, ok := h.(hook.MutateUpdateVirtual)
			if !ok {
				continue
			}

			res, err = m.MutateUpdateVirtual(ctx, res)
			if err != nil {
				return nil, fmt.Errorf("error mutating object: %v", err)
			}
		case "DeleteVirtual":
			m, ok := h.(hook.MutateDeleteVirtual)
			if !ok {
				continue
			}

			res, err = m.MutateDeleteVirtual(ctx, res)
			if err != nil {
				return nil, fmt.Errorf("error mutating object: %v", err)
			}
		case "GetVirtual":
			m, ok := h.(hook.MutateGetVirtual)
			if !ok {
				continue
			}

			res, err = m.MutateGetVirtual(ctx, res)
			if err != nil {
				return nil, fmt.Errorf("error mutating object: %v", err)
			}
		}

		rawObject, err := json.Marshal(res)
		if err != nil {
			return nil, fmt.Errorf("error encoding object %#+v: %v", res, err)
		}

		object = string(rawObject)
	}

	if object == originalObject {
		return &remote.MutateResult{}, nil
	}
	return &remote.MutateResult{Mutated: true, Object: object}, nil
}
