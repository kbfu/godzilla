package db

type Repo struct {
	Base
	Address string
	Name    string
	Branch  string
}

func (*Repo) TableName() string {
	return "repo"
}

func (r *Repo) FetchAll() (repos []Repo, err error) {
	tx := Db.Find(&repos)
	return repos, tx.Error
}
