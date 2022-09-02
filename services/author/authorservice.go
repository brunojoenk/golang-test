package services

import (
	"encoding/csv"
	"github/brunojoenk/golang-test/models/dtos"
	"github/brunojoenk/golang-test/models/entities"
	authorrepo "github/brunojoenk/golang-test/repository/author"
	"os"
	"sync"

	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

var BATCH_SIZE = 2000

type GetAllAuthors func(filter dtos.GetAuthorsFilter) ([]entities.Author, error)
type CreateAuthorInBatch func(author []entities.Author, batchSize int) error

type AuthorService struct {
	getAllAuthorsRepository GetAllAuthors
	createAuthorInBatchRepo CreateAuthorInBatch
}

// NewBookService Service Constructor
func NewAuthorService(db *gorm.DB) *AuthorService {
	repo := authorrepo.NewAuthorRepository(db)
	return &AuthorService{getAllAuthorsRepository: repo.GetAllAuthors, createAuthorInBatchRepo: repo.CreateAuthorInBatch}
}

func (a *AuthorService) GetAllAuthors(filter dtos.GetAuthorsFilter) (*dtos.AuthorResponseMetadata, error) {

	filter.Pagination.ValidValuesAndSetDefault()
	authors, err := a.getAllAuthorsRepository(filter)
	if err != nil {
		log.Error("Error on get all authors from repositoriy: ", err.Error())
		return nil, err
	}

	authorsResponse := make([]dtos.AuthorResponse, 0)
	for _, a := range authors {
		authorResponse := &dtos.AuthorResponse{
			Id:   a.Id,
			Name: a.Name,
		}
		authorsResponse = append(authorsResponse, *authorResponse)
	}

	authorResponseMetada := &dtos.AuthorResponseMetadata{
		Authors:    authorsResponse,
		Pagination: filter.Pagination,
	}

	return authorResponseMetada, nil
}

// Import all author using concurrence
func (a *AuthorService) ImportAuthorsFromCSVFile(file string) (int, error) {
	f, err := os.Open(file)

	if err != nil {
		log.Error("Error on open file: ", err.Error())
		return 0, err
	}

	defer f.Close()

	fcsv := csv.NewReader(f)
	fcsv.Comma = ';'

	numWorkers := 20
	jobs := make(chan []entities.Author, numWorkers)
	res := make(chan []entities.Author)

	var wg sync.WaitGroup
	worker := func(jobs <-chan []entities.Author, results chan<- []entities.Author) error {
		for {
			select {
			case job, ok := <-jobs: // you must check for readable state of the channel.
				if !ok {
					return nil
				}
				err := a.createAuthorInBatchRepo(job, len(job))
				if err != nil {
					log.Error("Error on create author in batch repository: ", err.Error())
					return err
				}
				results <- job
			}
		}
	}

	var errOnBatch error
	// init workers
	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func() {
			// this line will exec when chan `res` processed
			defer wg.Done()
			errOnBatch = worker(jobs, res)
		}()
	}

	go func() {
		mapper := make(map[string]bool, 0)
		rStr, err := fcsv.ReadAll()
		if err != nil {
			log.Error("Error on read all csv: ", err.Error())
			return
		}
		for _, record := range rStr {
			a.processRecord(record, mapper, jobs)
		}
		close(jobs) // close jobs to signal workers that no more job are incoming.
	}()

	go func() {
		wg.Wait()
		close(res) // when you close(res) it breaks the below loop.
	}()

	authors := make([]entities.Author, 0)
	for r := range res {
		authors = append(authors, r...)
	}

	return len(authors), errOnBatch
}

func (a *AuthorService) processRecord(record []string, mapper map[string]bool, jobs chan []entities.Author) {
	batch := make([]entities.Author, 0)
	for i, name := range record {
		if a.authorNotAdded(mapper, name) {
			mapper[name] = true
			batch = append(batch, entities.Author{Name: name})
		}
		if a.canCreateInBatch(i, len(record)) {
			jobs <- batch
			batch = make([]entities.Author, 0)
		}
	}
}

func (a *AuthorService) canCreateInBatch(index, recordSize int) bool {
	return (index > 0 && a.isCounterEqualBatchSize(index)) || a.isLastItemToIterate(index, recordSize)
}

func (a *AuthorService) isCounterEqualBatchSize(index int) bool {
	return index%BATCH_SIZE == 0
}

func (a *AuthorService) isLastItemToIterate(index, recordSize int) bool {
	return index == (recordSize - 1)
}

func (a *AuthorService) authorNotAdded(authorsAddedMap map[string]bool, name string) bool {
	return !authorsAddedMap[name]
}
