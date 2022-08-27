package main

type Questions struct {
	StatStatusPairs []struct {
		Stat StatRepo `json:"stat"`
	} `json:"stat_status_pairs"`
}

type StatRepo struct {
	QuestionID        int    `json:"question_id"`
	QuestionTitleSlug string `json:"question__title_slug"`
}

type LeetcodeGraphqlRequest struct {
	OperationName string `json:"operationName"`
	Variables     struct {
		TitleSlug string `json:"titleSlug"`
	} `json:"variables"`
	Query string `json:"query"`
}

type LeetCodeGraphqlResponse struct {
	Data struct {
		Question QuestionDetailRepo `json:"question"`
	} `json:"data"`
}

type QuestionDetailRepo struct {
	QuestionID        string             `json:"questionId"`
	TitleSlug         string             `json:"titleSlug"`
	Content           string             `json:"content"`
	TranslatedTitle   string             `json:"translatedTitle"`
	SampleTestCase    string             `json:"sampleTestCase"`
	TranslatedContent string             `json:"translatedContent"`
	CodeSnippets      []CodeSnippetsRepo `json:"codeSnippets"`
}

type CodeSnippetsRepo struct {
	Lang     string `json:"lang"`
	LangSlug string `json:"langSlug"`
	Code     string `json:"code"`
	Typename string `json:"__typename"`
}
