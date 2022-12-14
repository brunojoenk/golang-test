package services

import (
	"fmt"
	"github/brunojoenk/golang-test/models/dtos"
	"github/brunojoenk/golang-test/models/entities"
	authorrepo "github/brunojoenk/golang-test/repository/author"
	bookrepo "github/brunojoenk/golang-test/repository/book"
	"github/brunojoenk/golang-test/utils"

	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type IBookService interface {
	CreateBook(bookRequestCreate dtos.BookRequestCreate) error
	GetAllBooks(filter dtos.GetBooksFilter) (*dtos.BookResponseMetadata, error)
	DeleteBook(id int) error
	GetBook(id int) (*dtos.BookResponse, error)
	UpdateBook(id int, bookRequestUpdate dtos.BookRequestUpdate) error
}

type bookService struct {
	authorDb authorrepo.IAuthorRepository
	bookDb   bookrepo.IBookRepository
}

// NewBookService Service Constructor
func NewBookService(db *gorm.DB) IBookService {
	authorRepo := authorrepo.NewAuthorRepository(db)
	bookRepo := bookrepo.NewBookRepository(db)
	return &bookService{
		authorDb: authorRepo,
		bookDb:   bookRepo,
	}
}

func (b *bookService) CreateBook(bookRequestCreate dtos.BookRequestCreate) error {
	var authors []*entities.Author
	for _, authorId := range bookRequestCreate.Authors {
		author, err := b.authorDb.GetAuthor(authorId)
		if err != nil {
			log.Error("Error on get author from repo: ", err.Error())
			return err
		}
		if author.Id == 0 {
			return utils.ErrAuthorIdNotFound
		}
		authors = append(authors, author)
	}

	book := entities.Book{
		Name:            bookRequestCreate.Name,
		Edition:         bookRequestCreate.Edition,
		PublicationYear: bookRequestCreate.PublicationYear,
		Authors:         authors,
	}

	return b.bookDb.CreateBook(&book)
}

func (b *bookService) GetAllBooks(filter dtos.GetBooksFilter) (*dtos.BookResponseMetadata, error) {

	filter.Pagination.ValidValuesAndSetDefault()
	books, err := b.bookDb.GetAllBooks(filter)
	if err != nil {
		log.Error("Error on get all books from repo: ", err.Error())
		return nil, err
	}

	booksResponse := make([]dtos.BookResponse, 0)
	for _, book := range books {

		var authors string
		for i, author := range book.Authors {
			if i == 0 {
				authors = author.Name
				continue
			}
			authors += fmt.Sprintf(" | %s", author.Name)
		}

		bookResponse := &dtos.BookResponse{
			Id:              book.Id,
			Name:            book.Name,
			Edition:         book.Edition,
			PublicationYear: book.PublicationYear,
			Authors:         authors,
		}

		booksResponse = append(booksResponse, *bookResponse)
	}

	booksResponseMetadata := &dtos.BookResponseMetadata{
		Books:      booksResponse,
		Pagination: filter.Pagination,
	}

	return booksResponseMetadata, nil
}

func (b *bookService) DeleteBook(id int) error {
	return b.bookDb.DeleteBook(id)
}

func (b *bookService) GetBook(id int) (*dtos.BookResponse, error) {
	book, err := b.bookDb.GetBook(id)

	if err != nil {
		log.Error("Error on get book from repo: ", err.Error())
		return nil, err
	}

	if book.Id == 0 {
		return nil, utils.ErrBookIdNotFound
	}

	var authors string
	for i, author := range book.Authors {
		if i == 0 {
			authors = author.Name
			continue
		}
		authors += fmt.Sprintf(" | %s", author.Name)
	}

	bookResponse := dtos.BookResponse{
		Id:              book.Id,
		Name:            book.Name,
		Edition:         book.Edition,
		PublicationYear: book.PublicationYear,
		Authors:         authors,
	}

	return &bookResponse, nil
}

func (b *bookService) UpdateBook(id int, bookRequestUpdate dtos.BookRequestUpdate) error {
	book, err := b.bookDb.GetBook(id)

	if err != nil {
		log.Error("Error on get book from repo: ", err.Error())
		return err
	}

	var authors []*entities.Author
	for _, authorId := range bookRequestUpdate.Authors {
		author, err := b.authorDb.GetAuthor(authorId)
		if err != nil {
			log.Error("Error on get author from repo: ", err.Error())
			return err
		}
		if author.Id == 0 {
			return utils.ErrAuthorIdNotFound
		}
		authors = append(authors, author)
	}

	book.Name = bookRequestUpdate.Name
	book.Edition = bookRequestUpdate.Edition
	book.PublicationYear = bookRequestUpdate.PublicationYear

	err = b.bookDb.UpdateBook(book, authors)

	if err != nil {
		log.Error("Error on update book from repo: ", err.Error())
		return err
	}

	return nil
}
