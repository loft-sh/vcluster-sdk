package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"net/rpc"

	"github.com/hashicorp/go-plugin"
	"github.com/loft-sh/vcluster/pkg/plugin/types"
	v2 "github.com/loft-sh/vcluster/pkg/plugin/v2"
	"github.com/loft-sh/vcluster/pkg/plugin/v2/pluginv2"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
)

type server interface {
	plugin.Plugin

	// Serve starts the actual plugin server
	Serve()

	// SetReady signals the plugin server the plugin is ready to start
	SetReady(hooks map[types.VersionKindType][]ClientHook, interceptors []Interceptor, port int)

	// Initialized retrieves the initialize request
	Initialized() <-chan *pluginv2.Initialize_Request

	// IsLeader signals if syncer became leader
	IsLeader() <-chan struct{}
}

func newPluginServer() (server, error) {
	return &pluginServer{
		UnimplementedPluginServer: pluginv2.UnimplementedPluginServer{},

		initialized: make(chan *pluginv2.Initialize_Request),
		isReady:     make(chan struct{}),
		isLeader:    make(chan struct{}),
	}, nil
}

type pluginServer struct {
	pluginv2.UnimplementedPluginServer

	hooks            map[types.VersionKindType][]ClientHook
	interceptors     []Interceptor
	interceptorsPort int

	initialized chan *pluginv2.Initialize_Request
	isReady     chan struct{}
	isLeader    chan struct{}
}

var _ pluginv2.PluginServer = &pluginServer{}

func (p *pluginServer) Serve() {
	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: v2.HandshakeConfig,
		Plugins: map[string]plugin.Plugin{
			"plugin": p,
		},

		// A non-nil value here enables gRPC serving for this plugin...
		GRPCServer: plugin.DefaultGRPCServer,
	})
}

func (p *pluginServer) Initialize(ctx context.Context, initRequest *pluginv2.Initialize_Request) (*pluginv2.Initialize_Response, error) {
	// signal we can start up
	p.initialized <- initRequest

	// wait for plugin to become ready
	<-p.isReady

	// return back to syncer
	return &pluginv2.Initialize_Response{}, nil
}

func (p *pluginServer) Initialized() <-chan *pluginv2.Initialize_Request {
	return p.initialized
}

func (p *pluginServer) SetLeader(context.Context, *pluginv2.SetLeader_Request) (*pluginv2.SetLeader_Response, error) {
	close(p.isLeader)
	return &pluginv2.SetLeader_Response{}, nil
}

func (p *pluginServer) IsLeader() <-chan struct{} {
	return p.isLeader
}

func (p *pluginServer) SetReady(hooks map[types.VersionKindType][]ClientHook, interceptors []Interceptor, port int) {
	p.hooks = hooks
	p.interceptors = interceptors
	p.interceptorsPort = port
	close(p.isReady)
}

func (p *pluginServer) Mutate(ctx context.Context, req *pluginv2.Mutate_Request) (*pluginv2.Mutate_Response, error) {
	hooks, ok := p.hooks[types.VersionKindType{
		APIVersion: req.ApiVersion,
		Kind:       req.Kind,
		Type:       req.Type,
	}]
	if !ok {
		return &pluginv2.Mutate_Response{}, nil
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
			m, ok := h.(MutateCreatePhysical)
			if !ok {
				continue
			}

			res, err = m.MutateCreatePhysical(ctx, res)
			if err != nil {
				return nil, fmt.Errorf("error mutating object: %v", err)
			}
		case "UpdatePhysical":
			m, ok := h.(MutateUpdatePhysical)
			if !ok {
				continue
			}

			res, err = m.MutateUpdatePhysical(ctx, res)
			if err != nil {
				return nil, fmt.Errorf("error mutating object: %v", err)
			}
		case "DeletePhysical":
			m, ok := h.(MutateDeletePhysical)
			if !ok {
				continue
			}

			res, err = m.MutateDeletePhysical(ctx, res)
			if err != nil {
				return nil, fmt.Errorf("error mutating object: %v", err)
			}
		case "GetPhysical":
			m, ok := h.(MutateGetPhysical)
			if !ok {
				continue
			}

			res, err = m.MutateGetPhysical(ctx, res)
			if err != nil {
				return nil, fmt.Errorf("error mutating object: %v", err)
			}
		case "CreateVirtual":
			m, ok := h.(MutateCreateVirtual)
			if !ok {
				continue
			}

			res, err = m.MutateCreateVirtual(ctx, res)
			if err != nil {
				return nil, fmt.Errorf("error mutating object: %v", err)
			}
		case "UpdateVirtual":
			m, ok := h.(MutateUpdateVirtual)
			if !ok {
				continue
			}

			res, err = m.MutateUpdateVirtual(ctx, res)
			if err != nil {
				return nil, fmt.Errorf("error mutating object: %v", err)
			}
		case "DeleteVirtual":
			m, ok := h.(MutateDeleteVirtual)
			if !ok {
				continue
			}

			res, err = m.MutateDeleteVirtual(ctx, res)
			if err != nil {
				return nil, fmt.Errorf("error mutating object: %v", err)
			}
		case "GetVirtual":
			m, ok := h.(MutateGetVirtual)
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
		return &pluginv2.Mutate_Response{}, nil
	}
	return &pluginv2.Mutate_Response{Mutated: true, Object: object}, nil
}

func (p *pluginServer) GetPluginConfig(context.Context, *pluginv2.GetPluginConfig_Request) (*pluginv2.GetPluginConfig_Response, error) {
	clientHooks, err := p.getClientHooks()
	if err != nil {
		return nil, err
	}

	interceptorConfig := p.getInterceptorConfig()
	// build plugin config
	pluginConfig := &v2.PluginConfig{
		ClientHooks:  clientHooks,
		Interceptors: interceptorConfig,
	}

	// marshal plugin config
	pluginConfigRaw, err := json.Marshal(pluginConfig)
	if err != nil {
		return nil, fmt.Errorf("encode plugin config: %w", err)
	}

	return &pluginv2.GetPluginConfig_Response{Config: string(pluginConfigRaw)}, nil
}

func (p *pluginServer) getClientHooks() ([]*v2.ClientHook, error) {
	// transform hooks
	registeredHooks := []*v2.ClientHook{}
	for key := range p.hooks {
		hookFound := false
		for _, h := range registeredHooks {
			if h.APIVersion == key.APIVersion && h.Kind == key.Kind {
				found := false
				for _, t := range h.Types {
					if t == key.Type {
						found = true
						break
					}
				}
				if !found {
					h.Types = append(h.Types, key.Type)
				}

				hookFound = true
				break
			}
		}

		if !hookFound {
			registeredHooks = append(registeredHooks, &v2.ClientHook{
				APIVersion: key.APIVersion,
				Kind:       key.Kind,
				Types:      []string{key.Type},
			})
		}
	}

	return registeredHooks, nil
}

func (p *pluginServer) getInterceptorConfig() map[string][]v2.InterceptorRule {
	res := make(map[string][]v2.InterceptorRule)
	for _, interceptor := range p.interceptors {
		res[interceptor.Name()] = interceptor.InterceptionRules()
	}

	return res
}

var _ plugin.Plugin = &pluginServer{}

// Server always returns an error; we're only implementing the GRPCPlugin
// interface, not the Plugin interface.
func (p *pluginServer) Server(*plugin.MuxBroker) (interface{}, error) {
	return nil, errors.New("vcluster plugin only implements gRPC server")
}

// Client always returns an error; we're only implementing the GRPCPlugin
// interface, not the Plugin interface.
func (p *pluginServer) Client(*plugin.MuxBroker, *rpc.Client) (interface{}, error) {
	return nil, errors.New("vcluster plugin only implements gRPC server")
}

// GRPCClient always returns an error; we're only implementing the server half
// of the interface.
func (p *pluginServer) GRPCClient(_ context.Context, _ *plugin.GRPCBroker, _ *grpc.ClientConn) (interface{}, error) {
	return nil, errors.New("vcluster plugin only implements gRPC server")
}

// GRPCServer registers the gRPC provider server with the gRPC server that
// go-plugin is standing up.
func (p *pluginServer) GRPCServer(_ *plugin.GRPCBroker, s *grpc.Server) error {
	pluginv2.RegisterPluginServer(s, p)
	return nil
}
