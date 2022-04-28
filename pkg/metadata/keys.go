package metadata

import (
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/cache"
)

const (
	OwningClusterIndex              = "owning-cluster"
	OperatorManagedLabelKey         = "authzed.com/managed-by"
	OperatorManagedLabelValue       = "operator"
	OwnerLabelKey                   = "authzed.com/cluster"
	ComponentLabelKey               = "authzed.com/cluster-component"
	ComponentSpiceDBLabelValue      = "spicedb"
	ComponentMigrationJobLabelValue = "migration-job"
	ComponentServiceAccountLabel    = "spicedb-serviceaccount"
	ComponentRoleLabel              = "spicedb-role"
	ComponentServiceLabel           = "spicedb-service"
	ComponentRoleBindingLabel       = "spicedb-rolebinding"
	SpiceDBMigrationRequirementsKey = "authzed.com/spicedb-migration"
	SpiceDBConfigKey                = "authzed.com/spicedb-configuration"
	FieldManager                    = "spicedb-operator"
)

var (
	ForceOwned               = metav1.ApplyOptions{FieldManager: FieldManager, Force: true}
	ManagedDependentSelector = MustParseSelector(fmt.Sprintf("%s=%s", OperatorManagedLabelKey, OperatorManagedLabelValue))
)

func SelectorForComponent(owner, component string) labels.Selector {
	return labels.SelectorFromSet(LabelsForComponent(owner, component))
}

func LabelsForComponent(owner, component string) map[string]string {
	return map[string]string{
		OwnerLabelKey:           owner,
		ComponentLabelKey:       component,
		OperatorManagedLabelKey: OperatorManagedLabelValue,
	}
}

func GVRMetaNamespaceKeyer(gvr schema.GroupVersionResource, key string) string {
	return fmt.Sprintf("%s.%s.%s::%s", gvr.Resource, gvr.Version, gvr.Group, key)
}

func GVRMetaNamespaceKeyFunc(gvr schema.GroupVersionResource, obj interface{}) (string, error) {
	if d, ok := obj.(cache.DeletedFinalStateUnknown); ok {
		return d.Key, nil
	}
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		return "", err
	}
	return GVRMetaNamespaceKeyer(gvr, key), nil
}

func SplitGVRMetaNamespaceKey(key string) (gvr *schema.GroupVersionResource, namespace, name string, err error) {
	before, after, ok := strings.Cut(key, "::")
	if !ok {
		err = fmt.Errorf("error parsing key: %s", key)
		return
	}
	gvr, _ = schema.ParseResourceArg(before)
	if gvr == nil {
		err = fmt.Errorf("error parsing gvr from key: %s", before)
		return
	}
	namespace, name, err = cache.SplitMetaNamespaceKey(after)
	return
}

func GetClusterKeyFromLabel(in interface{}) ([]string, error) {
	obj := in.(runtime.Object)
	objMeta, err := meta.Accessor(obj)
	if err != nil {
		return nil, err
	}
	objLabels := objMeta.GetLabels()
	clusterName, ok := objLabels[OwnerLabelKey]
	if !ok {
		return nil, fmt.Errorf("synced %s %s/%s is managed by the operator but not associated with any cluster", obj.GetObjectKind(), objMeta.GetNamespace(), objMeta.GetName())
	}
	nn := types.NamespacedName{Name: clusterName, Namespace: objMeta.GetNamespace()}
	return []string{nn.String()}, nil
}