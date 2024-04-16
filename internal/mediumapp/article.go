package mediumapp

import (
	"errors"
	"example.com/medium/ent"
	"example.com/medium/ent/article"
	"example.com/medium/ent/user"
	"fmt"
	"github.com/labstack/echo/v4"
	"net/http"
	"strconv"
)

func InitArticleController(e *echo.Group, client *ent.Client) {
	controller := ArticleController{client}
	e.GET("/new-article", controller.NewArticle)
	e.GET("/article/:id", controller.GetArticle)
	e.POST("/new-article", controller.SaveNewArticle)
}

type ArticleController struct {
	*ent.Client
}

func (ArticleController) NewArticle(c echo.Context) error {
	userContext, err := GetUserContext(c)
	if err != nil && !errors.As(err, &jwtTokenMissingOrInvalid) {
		return err
	}
	errMap := make(map[string][]string)
	return render(c, http.StatusOK, newArticle(errMap, userContext))
}

func (controller ArticleController) GetArticle(c echo.Context) error {
	userContext, err := GetUserContext(c)
	if err != nil && !errors.As(err, &jwtTokenMissingOrInvalid) {
		return err
	}
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return render(c, http.StatusNotFound, notFound(UserContext{}))
	}
	articlePtr, err := controller.Client.Article.
		Query().WithUser().
		Where(article.ID(id)).
		Only(c.Request().Context())
	target := &ent.NotFoundError{}
	if errors.As(err, &target) {
		return render(c, http.StatusNotFound, notFound(userContext))
	}
	if err != nil {
		return fmt.Errorf("failed querying articles table: %w", err)
	}
	return render(c, http.StatusOK, articleDetailLayout(articlePtr, userContext))
}

type Article struct {
	Title   string `form:"title" validate:"required,gte=10"`
	Content string `form:"content" validate:"required,gte=20"`
}

func (controller ArticleController) SaveNewArticle(c echo.Context) error {
	userContext, err := GetUserContext(c)
	if err != nil && !errors.As(err, &jwtTokenMissingOrInvalid) {
		return err
	}
	userObj, queryErr := controller.Client.User.
		Query().
		Where(user.ID(userContext.ID)).
		Only(c.Request().Context())
	var notFound *ent.NotFoundError
	if errors.As(queryErr, &notFound) {
		errMap := map[string][]string{
			"nofield": {"User with such name and password doesn't exist."},
		}
		return render(c, http.StatusBadRequest, newArticle(errMap, UserContext{}))
	}
	articleForm := new(Article)

	if err := c.Bind(articleForm); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	errMap := GetErrorMap(validate.Struct(articleForm))
	if errMap != nil {
		return render(c, http.StatusBadRequest, newArticle(errMap, userContext))
	}

	_, saveErr := controller.Client.Article.
		Create().
		SetUserID(userObj.ID).
		SetTitle(articleForm.Title).
		SetContent(articleForm.Content).
		Save(c.Request().Context())
	if saveErr != nil {
		return saveErr
	}
	return c.Redirect(http.StatusSeeOther, "/")
}
