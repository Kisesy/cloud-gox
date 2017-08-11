package handler

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jpillora/cloud-gox/release"
)

type tagCompare struct {
	Commits []struct {
		Sha    string `json:"sha"`
		Commit struct {
			Author struct {
				Name  string    `json:"name"`
				Email string    `json:"email"`
				Date  time.Time `json:"date"`
			} `json:"author"`
			Committer struct {
				Name  string    `json:"name"`
				Email string    `json:"email"`
				Date  time.Time `json:"date"`
			} `json:"committer"`
			Message string `json:"message"`
			Tree    struct {
				Sha string `json:"sha"`
				URL string `json:"url"`
			} `json:"tree"`
			URL          string `json:"url"`
			CommentCount int    `json:"comment_count"`
		} `json:"commit"`
		URL         string `json:"url"`
		HTMLURL     string `json:"html_url"`
		CommentsURL string `json:"comments_url"`
		/*
			Author      struct {
				Login             string `json:"login"`
				ID                int    `json:"id"`
				AvatarURL         string `json:"avatar_url"`
				GravatarID        string `json:"gravatar_id"`
				URL               string `json:"url"`
				HTMLURL           string `json:"html_url"`
				FollowersURL      string `json:"followers_url"`
				FollowingURL      string `json:"following_url"`
				GistsURL          string `json:"gists_url"`
				StarredURL        string `json:"starred_url"`
				SubscriptionsURL  string `json:"subscriptions_url"`
				OrganizationsURL  string `json:"organizations_url"`
				ReposURL          string `json:"repos_url"`
				EventsURL         string `json:"events_url"`
				ReceivedEventsURL string `json:"received_events_url"`
				Type              string `json:"type"`
				SiteAdmin         bool   `json:"site_admin"`
			} `json:"author"`
			Committer struct {
				Login             string `json:"login"`
				ID                int    `json:"id"`
				AvatarURL         string `json:"avatar_url"`
				GravatarID        string `json:"gravatar_id"`
				URL               string `json:"url"`
				HTMLURL           string `json:"html_url"`
				FollowersURL      string `json:"followers_url"`
				FollowingURL      string `json:"following_url"`
				GistsURL          string `json:"gists_url"`
				StarredURL        string `json:"starred_url"`
				SubscriptionsURL  string `json:"subscriptions_url"`
				OrganizationsURL  string `json:"organizations_url"`
				ReposURL          string `json:"repos_url"`
				EventsURL         string `json:"events_url"`
				ReceivedEventsURL string `json:"received_events_url"`
				Type              string `json:"type"`
				SiteAdmin         bool   `json:"site_admin"`
			} `json:"committer"`
			Parents []struct {
				Sha     string `json:"sha"`
				URL     string `json:"url"`
				HTMLURL string `json:"html_url"`
			} `json:"parents"`
		*/
	} `json:"commits"`
	MergeBaseCommit struct {
		Sha    string `json:"sha"`
		Commit struct {
			Author struct {
				Name  string    `json:"name"`
				Email string    `json:"email"`
				Date  time.Time `json:"date"`
			} `json:"author"`
			Committer struct {
				Name  string    `json:"name"`
				Email string    `json:"email"`
				Date  time.Time `json:"date"`
			} `json:"committer"`
			Message string `json:"message"`
			Tree    struct {
				Sha string `json:"sha"`
				URL string `json:"url"`
			} `json:"tree"`
			URL          string `json:"url"`
			CommentCount int    `json:"comment_count"`
		} `json:"commit"`
		URL         string `json:"url"`
		HTMLURL     string `json:"html_url"`
		CommentsURL string `json:"comments_url"`
		Author      struct {
			Login             string `json:"login"`
			ID                int    `json:"id"`
			AvatarURL         string `json:"avatar_url"`
			GravatarID        string `json:"gravatar_id"`
			URL               string `json:"url"`
			HTMLURL           string `json:"html_url"`
			FollowersURL      string `json:"followers_url"`
			FollowingURL      string `json:"following_url"`
			GistsURL          string `json:"gists_url"`
			StarredURL        string `json:"starred_url"`
			SubscriptionsURL  string `json:"subscriptions_url"`
			OrganizationsURL  string `json:"organizations_url"`
			ReposURL          string `json:"repos_url"`
			EventsURL         string `json:"events_url"`
			ReceivedEventsURL string `json:"received_events_url"`
			Type              string `json:"type"`
			SiteAdmin         bool   `json:"site_admin"`
		} `json:"author"`
		Committer struct {
			Login             string `json:"login"`
			ID                int    `json:"id"`
			AvatarURL         string `json:"avatar_url"`
			GravatarID        string `json:"gravatar_id"`
			URL               string `json:"url"`
			HTMLURL           string `json:"html_url"`
			FollowersURL      string `json:"followers_url"`
			FollowingURL      string `json:"following_url"`
			GistsURL          string `json:"gists_url"`
			StarredURL        string `json:"starred_url"`
			SubscriptionsURL  string `json:"subscriptions_url"`
			OrganizationsURL  string `json:"organizations_url"`
			ReposURL          string `json:"repos_url"`
			EventsURL         string `json:"events_url"`
			ReceivedEventsURL string `json:"received_events_url"`
			Type              string `json:"type"`
			SiteAdmin         bool   `json:"site_admin"`
		} `json:"committer"`
		Parents []struct {
			Sha     string `json:"sha"`
			URL     string `json:"url"`
			HTMLURL string `json:"html_url"`
		} `json:"parents"`
	} `json:"merge_base_commit"`
}
type latestRelease struct {
	TagName         string `json:"tag_name"`
	TargetCommitish string `json:"target_commitish"`
}

var (
	GH_TOKEN string
)

func init() {
	GH_TOKEN = os.Getenv("GH_TOKEN")
}

func httpGet(url string) (data []byte, err error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return
	}
	req.Header.Set("Authorization", "token "+GH_TOKEN)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	data, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}
	return
}

func gen_desc(Package string) (desc string, err error) {
	ps := strings.Split(Package, "/")
	user, repo := ps[1], ps[2]

	// "https://api.github.com/repos/:owner/:repo/releases/latest"
	// apiurl := "https://api.github.com/repos/%s/%s/releases/latest"
	// GET /repos/:owner/:repo/releases
	apiurl := "https://api.github.com/repos/%s/%s/releases"
	apiurl = fmt.Sprintf(apiurl, user, repo)

	data, err := httpGet(apiurl)
	if err != nil {
		panic(err)
	}
	var lr []latestRelease
	err = json.Unmarshal(data, &lr)
	if err != nil {
		return
	}
	pre := 0
	if len(lr) > 1 {
		pre = 1
	}

	// lr.TagName = "1.2"

	// "GET /repos/:owner/:repo/compare/:base...:head"
	apiurl = "https://api.github.com/repos/%s/%s/compare/%s...%s"
	apiurl = fmt.Sprintf(apiurl, user, repo, lr[pre].TagName, lr[pre].TargetCommitish)
	data, err = httpGet(apiurl)
	if err != nil {
		return
	}
	var tc tagCompare
	err = json.Unmarshal(data, &tc)
	if err != nil {
		return
	}

	b := new(bytes.Buffer)
	b.WriteString("***ChangeLog:***\n")
	b.WriteString("---------\n")
	if len(tc.Commits) > 0 {
		for _, x := range tc.Commits {
			/*
				commit := x["commit"]
				committer := x["committer"]
				user_name = commit['author']['name']
				user_link = committer['html_url']
			*/
			commit_url := x.HTMLURL
			commit_hash := x.Sha[:7]
			message := x.Commit.Message
			// msg = '[{user_name}]({user_link}): [`{commit_sha1}`]({commit_url}) {message}'.format(**locals())
			b.WriteString(fmt.Sprintf("[`%s`](%s) %s\n", commit_hash, commit_url, message))
		}
	} else {
		mc := tc.MergeBaseCommit
		commit_url := mc.Parents[0].HTMLURL
		commit_hash := mc.Sha[:7]
		message := mc.Commit.Message
		// msg = '[{user_name}]({user_link}): [`{commit_sha1}`]({commit_url}) {message}'.format(**locals())
		b.WriteString(fmt.Sprintf("[`%s`](%s) %s\n", commit_hash, commit_url, message))
	}
	return b.String(), nil
}

//temporary storeage for the resulting binaries
var tempBuild = path.Join(os.TempDir(), "cloudgox")

//server's compile method
func (s *goxHandler) compile(c *Compilation) error {
	s.Printf("compiling %s...\n", c.Package)
	s.Printf("version %s\n", c.Version)
	c.StartedAt = time.Now()
	//optional releaser
	releaser := s.releasers[c.Releaser]
	var rel release.Release
	once := sync.Once{}
	setupRelease := func() {
		desc, err := gen_desc(c.Package)
		if err != nil {
			s.Printf("Warning: %s\n", err)
		}

		if r, err := releaser.Setup(c.Package, c.Version, desc); err == nil {
			rel = r
			s.Printf("%s successfully setup release %s (%s)\n", c.Releaser, c.Package, c.Version)
		} else {
			s.Printf("%s failed to setup release %s (%s)\n", c.Releaser, c.Package, err)
		}
	}
	//setup temp dir
	buildDir := filepath.Join(tempBuild, c.ID)
	if err := os.Mkdir(buildDir, 0755); err != nil && !os.IsExist(err) {
		return fmt.Errorf("Failed to create build directory %s", err)
	}
	pkgDir := filepath.Join(s.config.Path, "src", c.Package)
	//get target package
	if c.GoGet {
		if err := s.exec(".", "go", nil, "get", "-v", c.Package); err != nil {
			return fmt.Errorf("failed to get dependencies %s (%s)", c.Package, err)
		}
	}
	if _, err := os.Stat(pkgDir); err != nil {
		return fmt.Errorf("failed to find package %s", c.Package)
	}
	if c.Commitish != "" {
		s.Printf("loading specific commit %s\n", c.Commitish)
		//go to specific commit
		if err := s.exec(pkgDir, "git", nil, "status"); err != nil {
			return fmt.Errorf("failed to load commit: %s: %s is not a git repo", c.Commitish, c.Package)
		}
		if err := s.exec(pkgDir, "git", nil, "checkout", c.Commitish); err != nil {
			return fmt.Errorf("failed to load commit %s: %s", c.Package, err)
		}
		c.Variables[c.CommitVar] = c.Commitish
	} else {
		//commitish not set, attempt to find it
		s.Printf("retrieving current commit hash\n")
		cmd := exec.Command("git", "rev-parse", "--short", "HEAD")
		cmd.Dir = pkgDir
		if out, err := cmd.Output(); err == nil {
			currCommitish := strings.TrimSuffix(string(out), "\n")
			c.Variables[c.CommitVar] = currCommitish
		}
	}
	//calculate ldflags
	ldflags := []string{}
	if c.Shrink {
		s.Printf("ld-flag: -s -w (shrink)")
		ldflags = append(ldflags, "-s", "-w")
	}
	c.Variables["main.CLOUD_GOX"] = "1"
	c.Variables["main.BUILD_TIME"] = strconv.FormatInt(time.Now().Unix(), 10)
	for k, v := range c.Variables {
		s.Printf("ld-flag-X: %s=%s", k, v)
		ldflags = append(ldflags, "-X "+k+"="+v)
	}
	//compile all combinations of each target and each osarch
	for _, t := range c.Targets {
		target := filepath.Join(c.Package, t)
		targetDir := filepath.Join(pkgDir, t)
		targetName := filepath.Base(target)
		//go-get target deps
		if c.GoGet && targetDir != pkgDir {
			if err := s.exec(targetDir, "go", nil, "get", "-v", "."); err != nil {
				s.Printf("failed to get dependencies  of subdirectory %s", t)
				continue
			}
		}
		//compile target for all os/arch combos
		for _, osarchstr := range c.OSArch {
			osarch := strings.SplitN(osarchstr, "/", 2)
			osname := osarch[0]
			arch := osarch[1]
			targetFilename := fmt.Sprintf("%s_%s_%s", targetName, osname, arch)
			if osname == "windows" {
				targetFilename += ".exe"
			}
			targetOut := filepath.Join(buildDir, targetFilename)
			if _, err := os.Stat(targetDir); err != nil {
				s.Printf("failed to find target %s\n", target)
				continue
			}
			args := []string{
				"build",
				"-a",
				"-v",
				"-ldflags", strings.Join(ldflags, " "),
				"-o", targetOut,
				".",
			}
			c.Env["GOOS"] = osname
			c.Env["GOARCH"] = arch
			if !c.CGO {
				s.Printf("cgo disabled")
				c.Env["CGO_ENABLED"] = "0"
			}
			env := environ{}
			for k, v := range c.Env {
				s.Printf("env: %s=%s", k, v)
				env[k] = v
			}
			//run go build with cross compile configuration
			if err := s.exec(targetDir, "go", env, args...); err != nil {
				s.Printf("failed to build %s\n", targetFilename)
				continue
			}
			//gzip file
			b, err := ioutil.ReadFile(targetOut)
			if err != nil {
				return err
			}
			gzb := bytes.Buffer{}
			gz := gzip.NewWriter(&gzb)
			gz.Write(b)
			gz.Close()
			b = gzb.Bytes()
			targetFilename += ".gz"

			//optional releaser
			if releaser != nil {
				once.Do(setupRelease)
			}
			if rel != nil {
				if err := rel.Upload(targetFilename, b); err == nil {
					s.Printf("%s included asset in release %s\n", c.Releaser, targetFilename)
				} else {
					s.Printf("%s failed to release asset %s: %s\n", c.Releaser, targetFilename, err)
				}
			}
			//swap non-gzipd with gzipd
			if err := os.Remove(targetOut); err != nil {
				s.Printf("asset local remove failed %s\n", err)
				continue
			}
			targetOut += ".gz"
			if err := ioutil.WriteFile(targetOut, b, 0755); err != nil {
				s.Printf("asset local write failed %s\n", err)
				continue
			}
			//ready for download!
			s.Printf("compiled %s\n", targetFilename)
			c.Files = append(c.Files, targetFilename)
			s.state.Update()
		}
	}

	if c.Commitish != "" {
		s.Printf("revert repo back to latest commit\n")
		if err := s.exec(pkgDir, "git", nil, "checkout", "-"); err != nil {
			s.Printf("failed to revert commit %s: %s", c.Package, err)
		}
	}

	if len(c.Files) == 0 {
		return errors.New("No files compiled")
	}
	s.Printf("compiled %s (%s)\n", c.Package, c.Version)
	return nil
}
