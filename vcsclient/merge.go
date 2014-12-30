package vcsclient

import "sourcegraph.com/sourcegraph/go-vcs/vcs"

func (r *repository) MergeBase(a, b vcs.CommitID) (vcs.CommitID, error) {
	url, err := r.url(RouteRepoMergeBase, map[string]string{"CommitID1": string(a), "CommitID2": string(b)}, nil)
	if err != nil {
		return "", err
	}

	req, err := r.client.NewRequest("GET", url.String(), nil)
	if err != nil {
		return "", err
	}

	resp, err := r.client.doIgnoringRedirects(req)
	if err != nil {
		return "", err
	}

	return r.parseCommitIDInURL(resp.Header.Get("location"))
}
