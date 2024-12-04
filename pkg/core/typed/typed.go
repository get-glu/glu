package typed

import (
	"context"
	"fmt"
	"strings"

	"github.com/get-glu/glu/pkg/containers"
	"github.com/get-glu/glu/pkg/core"
)

var (
	KindUpdate    = "update"
	KindPromotion = "promotion"
	KindRollback  = "rollback"
)

// Phase is an interface around storage for resources.
type Phase[R core.Resource] interface {
	core.Phase
	GetResource(_ context.Context) (R, error)
}

// UpdatablePhase is a source through which the phase can promote resources to new versions
type UpdatablePhase[R core.Resource] interface {
	Phase[R]
	Update(_ context.Context, to R, opts ...containers.Option[UpdateOptions]) (*core.Result, error)
}

// UpdateOptions carries some context regarding the update
type UpdateOptions struct {
	Kind string
}

// NewUpdateOptions converts a list of UpdateOptions functional options
// into an instacne of *UpdateOptions.
func NewUpdateOptions(opts ...containers.Option[UpdateOptions]) *UpdateOptions {
	def := &UpdateOptions{Kind: KindUpdate}
	containers.ApplyAll(def, opts...)
	return def
}

// UpdateWithKind configures an UpdateOptions with a specific kind.
func UpdateWithKind(kind string) containers.Option[UpdateOptions] {
	return func(o *UpdateOptions) {
		o.Kind = kind
	}
}

func (o *UpdateOptions) Verb() string {
	switch o.Kind {
	case KindUpdate:
		return "update"
	case KindPromotion:
		return "promote"
	case KindRollback:
		return "rollback"
	default:
		return o.Kind
	}
}

// DefaultMessage returns the default description of the operations intent
// for a given phase
func (o *UpdateOptions) DefaultMessage(phase core.Descriptor) string {
	return fmt.Sprintf("%s %s", strings.ToTitle(o.Verb()), phase.Metadata.Name)
}
