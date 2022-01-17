package syncer

import (
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Managed is used to determine if a given physical object is managed by the virtual cluster
type Managed interface {
	// IsManaged determines if the given physical object is managed by the virtual cluster
	IsManaged(pObj client.Object) (bool, error)
}

// NameTranslator is used to translate an object name between the host cluster and virtual cluster
type NameTranslator interface {
	// VirtualToPhysical translates the given virtual name and optional object to the physical
	// name in the host cluster
	VirtualToPhysical(req types.NamespacedName, vObj client.Object) types.NamespacedName

	// PhysicalToVirtual translates the given physical name to the virtual cluster name
	PhysicalToVirtual(pObj client.Object) types.NamespacedName
}

// MetadataTranslator is used to translate an object's metadata, especially the annotations and labels
type MetadataTranslator interface {
	// TranslateMetadata initially copies the object and then translates the metadata
	TranslateMetadata(vObj client.Object) (client.Object, error)

	// TranslateMetadataUpdate translates the object metadata and then
	TranslateMetadataUpdate(vObj client.Object, pObj client.Object) map[string]string
}
