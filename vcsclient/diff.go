package vcsclient

import "sourcegraph.com/sourcegraph/go-vcs/vcs"

func (r *repository) Diff(base, head vcs.CommitID, opt *vcs.DiffOptions) (*vcs.Diff, error) {
	url, err := r.url(RouteRepoDiff, map[string]string{"Base": string(base), "Head": string(head)}, opt)
	if err != nil {
		return nil, err
	}

	req, err := r.client.NewRequest("GET", url.String(), nil)
	if err != nil {
		return nil, err
	}

	var diff *vcs.Diff
	if _, err := r.client.Do(req, &diff); err != nil {
		return nil, err
	}

	return diff, nil
}
