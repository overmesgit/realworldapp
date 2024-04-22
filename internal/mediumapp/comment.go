package mediumapp

import (
	"errors"
	"example.com/medium/ent"
	"example.com/medium/ent/user"
	"github.com/labstack/echo/v4"
	"net/http"
	"strconv"
)

func InitCommentController(e *echo.Group, client *ent.Client) {
	controller := CommentController{client}
	e.POST("/article/:id/new-comment", controller.SaveNewComment)
}

type CommentController struct {
	*ent.Client
}

type Comment struct {
	Text string `form:"text" validate:"required,gte=10"`
}

func (controller CommentController) SaveNewComment(c echo.Context) error {
	userContext, err := GetUserContext(c)
	if err != nil && !errors.As(err, &jwtTokenMissingOrInvalid) {
		return err
	}
	articleID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return render(c, http.StatusNotFound, notFound(UserContext{}))
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
		return render(c, http.StatusBadRequest, newComment(articleID, nil, nil, errMap))
	}
	commentForm := new(Comment)

	if err := c.Bind(commentForm); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	errMap := GetErrorMap(validate.Struct(commentForm))
	if errMap != nil {
		return render(c, http.StatusOK, newComment(articleID, nil, nil, errMap))
	}

	comment, saveErr := controller.Client.Comment.
		Create().
		SetArticleID(articleID).
		SetUser(userObj).
		SetText(commentForm.Text).
		Save(c.Request().Context())
	if saveErr != nil {
		return saveErr
	}
	emptyErrMap := make(map[string][]string)
	return render(c, http.StatusOK, newComment(articleID, comment, userObj, emptyErrMap))
}
