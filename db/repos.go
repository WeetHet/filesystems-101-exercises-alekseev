package db

import (
	"tests-runner/util/errors"
	"tests-runner/util/ulid"
)

type Repo struct {
	Id  ulid.ULID `json:"id"`
	Url string    `json:"url"`
}

func (db *DB) GetRepoByURL(url string) (r *Repo, err error) {
	defer func() { err = errors.Wrapf(err, "GetRepoByURL(%q)", url) }()

	r = &Repo{}
	if err = db.orm.Where("url = ?", url).First(r).Error; err != nil {
		return nil, err
	}
	return r, nil
}

func (db *DB) GetRepoById(id ulid.ULID) (r *Repo, err error) {
	defer func() { err = errors.Wrapf(err, "GetRepoById(%s)", id) }()

	r = &Repo{}
	if err = db.orm.Where("id = ?", id).First(r).Error; err != nil {
		return nil, err
	}
	return r, nil
}

func (db *DB) InsertRepo(r *Repo) (err error) {
	defer func() { err = errors.Wrapf(err, "InsertRepo(%q)", r.Url) }()

	return db.orm.Create(r).Error
}
