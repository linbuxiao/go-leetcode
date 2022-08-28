package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"io/ioutil"
	"strings"

	"github.com/go-resty/resty/v2"
	"github.com/spf13/afero"
)

// request url
const BASE_URL = "https://leetcode.cn"
const QUESTIONS_URL = "/api/problems/algorithms/"
const GRAPHQL_URL = "/graphql/"
const PROBLEM_PREFIX = "/problems"

// request query
const QUESTION_QUERY = "query questionData($titleSlug: String!) {\n  question(titleSlug: $titleSlug) {\n    questionId\n    titleSlug\n    content\n    translatedTitle\n    sampleTestCase\n   translatedContent\n    codeSnippets {\n      lang\n      langSlug\n      code\n      __typename\n    }\n}\n}\n"

// template
const TEMPLATE_FILE_URL = "template/question.tmpl"
const TEMPLATE_QUESTION_DESCRIPTION = "__QUESTION_DESCRIPTION__"
const TEMPLATE_QUESTION_LINK = "__QUESTION_LINK__"
const TEMPLATE_PACKAGE_NAME = "__PACKAGE_NAME__"
const TEMPLATE_CODE = "__CODE__"

// static error
var ErrCannotFindQuestionByID = errors.New("cannot find question by id")
var ErrCannotGetGoSnippets = errors.New("cannot get golang code snippets")

type Question struct {
	client *resty.Client
	ID     int
	Slug   string
	Detail *QuestionDetailRepo
}

func NewQuestion(id int) *Question {
	client := resty.New().SetBaseURL(BASE_URL)
	return &Question{
		client: client,
		ID:     id,
	}
}

func (q *Question) Build() error {
	allQuestion, err := q.getAllQuestion()
	if err != nil {
		return err
	}
	stat := q.getQuestionStatByID(allQuestion)
	if stat == nil {
		return ErrCannotFindQuestionByID
	}
	q.Slug = stat.QuestionTitleSlug
	return q.setQuestionDetail()
}

func (q *Question) getAllQuestion() (*Questions, error) {
	resp, err := q.client.R().Get(QUESTIONS_URL)
	if err != nil {
		return nil, err
	}
	var all Questions
	err = json.Unmarshal(resp.Body(), &all)
	return &all, err
}

func (q *Question) getQuestionStatByID(questions *Questions) *StatRepo {
	for _, x := range questions.StatStatusPairs {
		if x.Stat.QuestionID == q.ID {
			return &x.Stat
		}
	}
	return nil
}

func (q *Question) setQuestionDetail() error {
	req := LeetcodeGraphqlRequest{
		OperationName: "questionData",
		Query:         QUESTION_QUERY,
		Variables: struct {
			TitleSlug string "json:\"titleSlug\""
		}{
			TitleSlug: q.Slug,
		},
	}
	var res LeetCodeGraphqlResponse
	_, err := q.client.R().SetBody(&req).SetResult(&res).Post(GRAPHQL_URL)
	if err != nil {
		return err
	}
	q.Detail = &res.Data.Question
	return nil
}

type QuestionFile struct {
	Question *Question
	os       afero.Fs
}

func NewQuestionFile(q *Question) *QuestionFile {
	return &QuestionFile{Question: q, os: afero.NewOsFs()}
}

func (f *QuestionFile) getPackageName() string {
	fmtTitle := strings.ReplaceAll(f.Question.Slug, "-", "_")
	return fmt.Sprintf("s%d_%s", f.Question.ID, fmtTitle)
}

func (f *QuestionFile) loadQuestionTemplate() (string, error) {
	file, err := f.os.Open(TEMPLATE_FILE_URL)
	if err != nil {
		return "", err
	}
	b, err := ioutil.ReadAll(file)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

type desc string

func (d desc) replace(old, new string) desc {
	r := strings.ReplaceAll(string(d), old, new)
	return desc(r)
}

func (f *QuestionFile) buildDesc() string {
	r := desc(f.Question.Detail.TranslatedContent).
		replace("<strong>", "").
		replace("</strong>", "").
		replace("<em>", "").
		replace("</em>", "").
		replace("</p>", "").
		replace("<p>", "").
		replace("<b>", "").
		replace("</b>", "").
		replace("<pre>", "").
		replace("</pre>", "").
		replace("<ul>", "").
		replace("</ul>", "").
		replace("<li>", "").
		replace("</li>", "").
		replace("<code>", "").
		replace("</code>", "").
		replace("<i>", "").
		replace("</i>", "").
		replace("<sub>", "").
		replace("</sub>", "").
		replace("</sup>", "").
		replace("<sup>", "^").
		replace("&nbsp;", " ").
		replace("&gt;", ">").
		replace("&lt;", "<").
		replace("&quot;", "\"").
		replace("&minus;", "-").
		replace("&#39;", "'").
		replace("\n\n", "\n")
	return string(r)
}

func (f *QuestionFile) buildCode() (string, error) {
	for _, c := range f.Question.Detail.CodeSnippets {
		if c.Lang == "Go" {
			return c.Code, nil
		}
	}
	return "", ErrCannotGetGoSnippets
}

func (f *QuestionFile) injectToDo(code string) string {
	buf := bytes.NewBuffer(nil)
	fset := token.NewFileSet()
	astFile, err := parser.ParseFile(fset, "", code, parser.ParseComments)
	if err != nil {
		panic(err)
	}
	ast.Inspect(astFile, func(n ast.Node) bool {
		expr, ok := n.(*ast.FuncDecl)
		if !ok {
			return true
		}
		e, err := parser.ParseExpr(`panic("TODO")`)
		if err != nil {
			panic(err)
		}
		expr.Body.List = []ast.Stmt{
			&ast.ExprStmt{
				X: e,
			},
		}
		return true
	})
	if err := format.Node(buf, fset, astFile); err != nil {
		panic(err)
	}
	return buf.String()
}

func (f *QuestionFile) Create() error {
	// prepare
	questionTemplate, err := f.loadQuestionTemplate()
	if err != nil {
		return err
	}
	desc := f.buildDesc()
	code, err := f.buildCode()
	if err != nil {
		return err
	}
	packageName := f.getPackageName()
	// create file
	// 1. render file
	content := strings.ReplaceAll(questionTemplate, TEMPLATE_QUESTION_DESCRIPTION, desc)
	content = strings.ReplaceAll(content, TEMPLATE_QUESTION_LINK, fmt.Sprintf("%s%s/%s", BASE_URL, PROBLEM_PREFIX, f.Question.Slug))
	content = strings.ReplaceAll(content, TEMPLATE_CODE, code)
	content = strings.ReplaceAll(content, TEMPLATE_PACKAGE_NAME, packageName)
	content = f.injectToDo(content)
	// 2. create package
	if err := f.os.Mkdir(packageName, 0755); err != nil {
		return err
	}
	// 3. create
	q, err := f.os.Create(fmt.Sprintf("%s/%s.go", packageName, packageName))
	if err != nil {
		return err
	}
	// 4. write
	_, err = q.WriteString(content)
	return err
}
