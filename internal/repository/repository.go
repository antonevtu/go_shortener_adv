//Package repository implements in-memory entity storage
//Implements handlers.Repositorier interface, but some methods not supported (because this is education application)
//Storage has backup in text file cfgApp.FileStoragePath
package repository

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/antonevtu/go_shortener_adv/internal/db"
	"github.com/antonevtu/go_shortener_adv/internal/pool"
	"io"
	"os"
	"sync"
)

//Repository is in-memory repository, based on map, with backup file writer for new records
type Repository struct {
	storage     storageT
	storageLock sync.Mutex
	fileWriter  fileWriterT
}

type storageT map[string]db.Entity

type fileWriterT struct {
	file    *os.File
	encoder *json.Encoder
}

//New returns new in-memory repository, restored from text file
func New(fileName string) (*Repository, error) {
	repository := Repository{
		storage:    make(storageT, 100),
		fileWriter: fileWriterT{},
	}

	err := repository.restoreFromFile(fileName)
	if err != nil {
		return &repository, err
	}

	err = repository.fileWriter.new(fileName)
	if err != nil {
		return &repository, err
	}
	return &repository, nil
}

func (fw *fileWriterT) new(filename string) error {
	file, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0777)
	if err != nil {
		return err
	}
	*fw = fileWriterT{
		file:    file,
		encoder: json.NewEncoder(file),
	}
	return nil
}

func (r *Repository) restoreFromFile(fileName string) error {
	file, err := os.OpenFile(fileName, os.O_RDONLY|os.O_CREATE, 0777)
	if err != nil {
		return err
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	entity := db.Entity{}
	for {
		err = decoder.Decode(&entity)
		if err == io.EOF {
			return nil
		} else if err != nil {
			return err
		}
		r.storage[entity.ShortID] = entity
	}
}

func (r *Repository) AddEntity(_ context.Context, entity db.Entity) error {
	r.storageLock.Lock()
	defer r.storageLock.Unlock()
	r.storage[entity.ShortID] = entity
	err := r.fileWriter.encoder.Encode(&entity)
	return err
}

func (r *Repository) SelectByLongURL(_ context.Context, id string) (db.Entity, error) {
	return db.Entity{}, errors.New("method not supported")
}

func (r *Repository) SelectByShortID(_ context.Context, id string) (db.Entity, error) {
	r.storageLock.Lock()
	defer r.storageLock.Unlock()
	entity, ok := r.storage[id]
	if ok {
		return entity, nil
	} else {
		return db.Entity{}, errors.New("a non-existent ID was requested")
	}
}

func (r *Repository) SelectByUser(_ context.Context, userID string) ([]db.Entity, error) {
	r.storageLock.Lock()
	defer r.storageLock.Unlock()
	selection := make([]db.Entity, 0, 10)
	for _, entity := range r.storage {
		if userID == entity.UserID {
			selection = append(selection, entity)
		}
	}
	return selection, nil
}

func (r *Repository) Close() {
	_ = r.fileWriter.file.Close()
}

func (r *Repository) AddEntityBatch(_ context.Context, _ string, _ db.BatchInput) error {
	return errors.New("batches not supported")
}

func (r *Repository) Ping(_ context.Context) error {
	return errors.New("ping not supported")
}

func (r *Repository) SetDeletedBatch(ctx context.Context, userID string, shortIDs []string) error {
	return errors.New("method not supported")
}

func (r *Repository) SetDeleted(ctx context.Context, item pool.ToDeleteItem) error {
	return errors.New("method not supported")
}
