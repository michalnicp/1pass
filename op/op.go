package op

import (
	"encoding/json"
	"io"
	"log"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"

	"github.com/blevesearch/bleve"
	"github.com/blevesearch/bleve/search/query"
	"github.com/blevesearch/bleve/search/searcher"
	cache "github.com/patrickmn/go-cache"
	"github.com/pkg/errors"
)

// defaultSigninAddress is the default 1password signin address for a personal account.
const defaultSigninAddress = "my.1password.com"

func init() {
	searcher.MaxFuzziness = 5
}

// Session represents a 1Password session.
type Session struct {
	SigninAddress string
	Email         string
	SecretKey     string
	Token         string
	expiry        time.Time

	cache *cache.Cache
	index bleve.Index
}

// NewSession creates a new 1password session.
func NewSession(signinAddress, email, secretKey, token string) (*Session, error) {
	cache := cache.New(15*time.Minute, 5*time.Minute)

	// TODO: Improve search.
	mapping := bleve.NewIndexMapping()
	index, err := bleve.NewMemOnly(mapping)
	if err != nil {
		return nil, errors.Wrap(err, "create index")
	}

	session := Session{
		SigninAddress: signinAddress,
		Email:         email,
		SecretKey:     secretKey,
		Token:         token,
		cache:         cache,
		index:         index,
	}

	return &session, nil
}

func NewSessionFromConfig() (*Session, error) {
	cfg, err := ReadConfig()
	if err != nil {
		return nil, errors.Wrap(err, "read op config")
	}

	for _, account := range cfg.Accounts {
		if account.Shorthand == cfg.LatestSignin {
			name := "OP_SESSION_" + cfg.LatestSignin
			token := os.Getenv(name)
			if token == "" {
				log.Printf("session token not found: %s is empty", name)
			}

			session, err := NewSession(account.URL, account.Email, account.AccountKey, token)
			if err != nil {
				return nil, errors.Wrap(err, "create session")
			}

			// Check if session is valid.
			cmd := exec.Command("op", "get", "account", "--session="+session.Token)

			if _, err := cmd.Output(); err != nil {
				return nil, fromExitError(err)
			}

			session.expiry = time.Now().Add(30 * time.Minute)

			return session, nil
		}
	}

	return nil, errors.WithStack(ErrInvalidOPConfig)
}

// Signin signs in with 1Password and returns a session.
func Signin(signinAddress, email, secretKey, masterPassword string) (*Session, error) {
	if signinAddress == "" {
		signinAddress = defaultSigninAddress
	}

	cmd := exec.Command("op", "signin", signinAddress, email, secretKey, masterPassword, "--output=raw")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, errors.Wrap(err, "open stdin pipe")
	}

	go func() {
		defer stdin.Close()
		io.WriteString(stdin, masterPassword)
	}()

	out, err := cmd.Output()
	if err != nil {
		return nil, fromExitError(err)
	}
	token := strings.TrimSpace(string(out))

	session, err := NewSession(signinAddress, email, secretKey, token)
	if err != nil {
		return nil, errors.Wrap(err, "create session")
	}

	return session, nil
}

func (s *Session) Valid() bool {
	if s == nil || s.Token == "" {
		return false
	}
	if time.Now().After(s.expiry) {
		return false
	}
	return true
}

type Item struct {
	UUID     string `json:"uuid"`
	Overview struct {
		Title string `json:"title"`
		AInfo string `json:"ainfo"`
	} `json:"overview"`
	Details *Details `json:"details,omitempty"` // omitted when listing items
}

type Details struct {
	Fields   []DetailsField `json:"fields"`
	Notes    string         `json:"notesPlain"`
	Sections []Section      `json:"sections"`
}

type DetailsField struct {
	Designation string `json:"designation"`
	Name        string `json:"name"`
	Value       string `json:"value"`
}

type Section struct {
	Name   string         `json:"name"`
	Title  string         `json:"title"`
	Fields []SectionField `json:"fields"`
}

type SectionField struct {
	Type  string `json:"k"`
	Title string `json:"t"`
	Value string `json:"v"`
}

func (s *Session) ListItems() ([]Item, error) {
	if items, ok := s.cache.Get("items"); ok {
		return items.([]Item), nil
	}

	cmd := exec.Command("op", "list", "items", "--session="+s.Token)

	out, err := cmd.Output()
	if err != nil {
		return nil, fromExitError(err)
	}

	var items []Item
	if err := json.Unmarshal(out, &items); err != nil {
		return nil, err
	}

	// Sort the items by uuid.
	sort.Slice(items, func(i, j int) bool {
		return items[i].UUID < items[j].UUID
	})

	// Store the items in the cache using default expiry.
	s.cache.SetDefault("items", items)

	// Index the items for searching.
	t := time.Now()

	batch := s.index.NewBatch()
	for _, item := range items {
		batch.Index(item.UUID, item)
	}

	if err := s.index.Batch(batch); err != nil {
		return nil, errors.Wrap(err, "index items")
	}

	log.Printf("index took %s", time.Since(t))

	return items, nil
}

func (s *Session) SearchItems(queryStr string) ([]Item, error) {
	items, err := s.ListItems()
	if err != nil {
		return nil, errors.Wrap(err, "list items")
	}

	if queryStr == "" {
		return items, nil
	}

	// TODO: Improve search.
	var disjuncts []query.Query
	{
		query := bleve.NewFuzzyQuery(queryStr)
		query.SetFuzziness(2)
		query.SetBoost(1)
		disjuncts = append(disjuncts, query)
	}
	{
		query := bleve.NewPrefixQuery(queryStr)
		query.SetBoost(4)
		disjuncts = append(disjuncts, query)
	}
	query := query.NewDisjunctionQuery(disjuncts)

	searchRequest := bleve.NewSearchRequest(query)
	searchRequest.Fields = []string{"overview.title"}
	searchResults, err := s.index.Search(searchRequest)
	if err != nil {
		return nil, errors.Wrap(err, "search index")
	}

	results := make([]Item, len(searchResults.Hits))
	for i, match := range searchResults.Hits {
		j := sort.Search(len(items), func(k int) bool {
			return items[k].UUID >= match.ID
		})
		results[i] = items[j]
	}

	return results, nil
}

func (s *Session) GetItem(id string) (*Item, error) {
	if item, ok := s.cache.Get("item:" + id); ok {
		return item.(*Item), nil
	}

	cmd := exec.Command("op", "get", "item", id, "--session="+s.Token)

	out, err := cmd.Output()
	if err != nil {
		return nil, fromExitError(err)
	}

	var item Item
	if err := json.Unmarshal(out, &item); err != nil {
		return nil, err
	}

	// Store the item in the cache using default expiry.
	s.cache.SetDefault("item:"+id, &item)

	return &item, nil
}
