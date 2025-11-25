package articlesapi

// todo: for less allocation better to unmarshal only that fields
// that acting a some role(avoid redundant)

type (
	Response struct {
		Page       int      `json:"page"`
		PerPage    int      `json:"per_page"`
		Total      int      `json:"total"`
		TotalPages int      `json:"total_pages"`
		Data       Articles `json:"data"`
	}
	Article struct {
		Title       *string `json:"title"`
		URL         string  `json:"url"`
		Author      string  `json:"author"`
		NumComments *int    `json:"num_comments"`
		StoryID     *int    `json:"story_id"`
		StoryTitle  *string `json:"story_title"`
		StoryURL    *string `json:"story_url"`
		ParentID    *int    `json:"parent_id"`
		CreatedAt   *int    `json:"created_at"`
	}
	Articles []*Article
)
