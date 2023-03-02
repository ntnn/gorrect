package engines

import (
	"context"

	"github.com/ntnn/gorrect"
	"golang.org/x/exp/slog"
)

type Simple struct {
	elements map[string]gorrect.Identitier
	state    *simpleState
	log      *slog.Logger
}

func NewSimple(logger *slog.Logger) *Simple {
	return &Simple{
		elements: map[string]gorrect.Identitier{},
		log:      logger,
	}
}

func (simple *Simple) Add(elems ...gorrect.Identitier) error {
	for _, elem := range elems {
		ident := gorrect.FormatIdentitier(elem)
		l := simple.log.With(slog.String("element", ident))

		if validator, ok := elem.(gorrect.Validator); ok {
			l.Debug("validating")
			if err := validator.Validate(); err != nil {
				return err
			}
		}

		if baseIdentity, _ := elem.Identity(); baseIdentity != gorrect.Noop {
			if _, ok := simple.elements[ident]; ok {
				l.Error("element with same identity already exists", nil)
				return ErrSameIdentityAlreadyRegistered
			}
		}

		l.Debug("adding element")
		simple.elements[ident] = elem
	}
	return nil
}

func (simple Simple) Run(ctx context.Context) error {
	simple.state = newSimpleState()

	for {
		if len(simple.state.done) == len(simple.elements) {
			simple.log.Info("all elements are done")
			return nil
		}

		for _, elem := range simple.elements {
			ident := gorrect.FormatIdentitier(elem)
			l := simple.log.With(slog.Any("element", ident))

			// skip done elements
			if _, ok := simple.state.done[ident]; ok {
				continue
			}
			l.Debug("handling element")

			if !simple.preElementsDone(elem) {
				l.Debug("pre elements of element are not done")
				continue
			}

			executor, ok := elem.(gorrect.Executor)
			if !ok {
				simple.state.done[ident] = true
				continue
			}

			l.Info("executing element")
			if err := executor.Execute(ctx); err != nil {
				return err
			}

			simple.state.done[ident] = true
		}
	}
}

func (simple Simple) hasElems(elems ...string) bool {
	for _, elem := range elems {
		if _, ok := simple.elements[elem]; !ok {
			return false
		}
	}
	return true
}

func (simple Simple) preElementsDone(elem any) bool {
	preElementer, ok := elem.(gorrect.PreElementer)
	if !ok {
		return true
	}

	for _, preElement := range preElementer.PreElements() {
		if !simple.hasElems(preElement) {
			continue
		}

		done, ok := simple.state.done[preElement]
		if !ok {
			return false
		}

		if !done {
			return false
		}
	}

	return true
}
