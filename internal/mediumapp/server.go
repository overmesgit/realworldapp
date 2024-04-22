package mediumapp

import (
	"errors"
	"example.com/medium/ent"
	"example.com/medium/ent/article"
	"github.com/a-h/templ"
	"github.com/go-playground/validator/v10"
	"github.com/golang-jwt/jwt/v5"
	echojwt "github.com/labstack/echo-jwt/v4"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"net/http"
	"strconv"
)

var validate = validator.New(validator.WithRequiredStructEnabled())

type CustomValidator struct {
	validator *validator.Validate
}

func (cv *CustomValidator) Validate(i interface{}) error {
	if err := cv.validator.Struct(i); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	return nil
}

func render(ctx echo.Context, status int, t templ.Component) error {
	ctx.Response().Writer.WriteHeader(status)

	err := t.Render(ctx.Request().Context(), ctx.Response().Writer)
	if err != nil {
		return ctx.String(http.StatusInternalServerError, "failed to render response template")
	}

	return nil
}

type IndexController struct {
	*ent.Client
}

func (controller IndexController) IndexPage(c echo.Context) error {
	pageParam := c.Param("page")
	println(pageParam)
	page := 1
	if len(pageParam) > 0 {
		var err error
		page, err = strconv.Atoi(pageParam)
		if err != nil {
			return render(c, http.StatusNotFound, notFound(UserContext{}))
		}
	}
	userContext, err := GetUserContext(c)
	if err != nil && !errors.As(err, &jwtTokenMissingOrInvalid) {
		return err
	}
	articlesOnPage := 50
	articles, err := controller.Client.Article.
		Query().WithUser().Order(ent.Desc(article.FieldID)).
		Offset((page - 1) * articlesOnPage).Limit(articlesOnPage).
		All(c.Request().Context())
	if err != nil {
		return err
	}
	articlesCount, err := controller.Client.Article.
		Query().Order(ent.Desc(article.FieldID)).
		Count(c.Request().Context())
	if err != nil {
		return err
	}

	return index(articles, userContext, Pagination{page, (articlesCount / articlesOnPage) + 1}).
		Render(c.Request().Context(), c.Response().Writer)
}

func StartServer(client *ent.Client) {
	insecure := echo.New()
	insecure.Use(middleware.Logger())
	insecure.Static("/static", "assets")

	InitAuthController(insecure, client)
	secure := insecure.Group("")
	config := echojwt.Config{
		ContinueOnIgnoredError: true,
		ErrorHandler: func(c echo.Context, err error) error {
			return nil
		},
		NewClaimsFunc: func(c echo.Context) jwt.Claims {
			return new(jwtCustomClaims)
		},
		SigningKey:  AuthSecret,
		TokenLookup: "cookie:user",
	}
	secure.Use(echojwt.WithConfig(config))

	indexCtr := IndexController{client}
	secure.GET("/", indexCtr.IndexPage)
	secure.GET("/page/:page", indexCtr.IndexPage)

	secure.GET("/user", func(c echo.Context) error {
		user, err := getJwtClaims(c)

		if err != nil {
			return err
		}

		return c.JSON(http.StatusOK, user)
	})

	InitArticleController(secure, client)
	InitCommentController(secure, client)
	insecure.Logger.Fatal(insecure.Start(":8080"))
}
