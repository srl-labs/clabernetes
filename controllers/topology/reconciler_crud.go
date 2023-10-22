package topology

import (
	"context"

	apimachinerytypes "k8s.io/apimachinery/pkg/types"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	ctrlruntimeutil "sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func (r *Reconciler) createObj(
	ctx context.Context,
	ownerObj,
	renderedObj ctrlruntimeclient.Object,
) error {
	err := ctrlruntimeutil.SetOwnerReference(ownerObj, renderedObj, r.Client.Scheme())
	if err != nil {
		return err
	}

	r.Log.Debugf(
		"creating %s '%s/%s'",
		renderedObj.GetObjectKind().GroupVersionKind().Kind,
		renderedObj.GetNamespace(),
		renderedObj.GetName(),
	)

	err = r.Client.Create(ctx, renderedObj)
	if err != nil {
		r.Log.Criticalf(
			"failed creating %s '%s/%s' error: %s",
			renderedObj.GetObjectKind().GroupVersionKind().Kind,
			renderedObj.GetNamespace(),
			renderedObj.GetName(),
			err,
		)

		return err
	}

	return nil
}

func (r *Reconciler) getObj(
	ctx context.Context,
	getObj ctrlruntimeclient.Object,
	namespacedName apimachinerytypes.NamespacedName,
) error {
	r.Log.Debugf(
		"getting %s '%s/%s'",
		getObj.GetObjectKind().GroupVersionKind().Kind,
		getObj.GetNamespace(),
		getObj.GetName(),
	)

	return r.Client.Get(ctx, namespacedName, getObj)
}

func (r *Reconciler) updateObj(
	ctx context.Context,
	updateObj ctrlruntimeclient.Object,
) error {
	r.Log.Debugf(
		"updating %s '%s/%s'",
		updateObj.GetObjectKind().GroupVersionKind().Kind,
		updateObj.GetNamespace(),
		updateObj.GetName(),
	)

	err := r.Client.Update(ctx, updateObj)
	if err != nil {
		r.Log.Criticalf(
			"failed updating %s '%s/%s' error: %s",
			updateObj.GetObjectKind().GroupVersionKind().Kind,
			updateObj.GetNamespace(),
			updateObj.GetName(),
			err,
		)

		return err
	}

	return nil
}

func (r *Reconciler) deleteObj(
	ctx context.Context,
	deleteObj ctrlruntimeclient.Object,
) error {
	r.Log.Debugf(
		"deleting %s '%s/%s'",
		deleteObj.GetObjectKind().GroupVersionKind().Kind,
		deleteObj.GetNamespace(),
		deleteObj.GetName(),
	)

	err := r.Client.Delete(ctx, deleteObj)
	if err != nil {
		r.Log.Criticalf(
			"failed deleting %s '%s/%s' error: %s",
			deleteObj.GetObjectKind().GroupVersionKind().Kind,
			deleteObj.GetNamespace(),
			deleteObj.GetName(),
			err,
		)

		return err
	}

	return nil
}
