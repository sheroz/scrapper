package main

// https://github.com/GoogleCloudPlatform/golang-samples/blob/master/translate/snippets/snippet.go
// http://www.wadewegner.com/2014/12/easy-go-programming-setup-for-windows/

import (
	"fmt"
	"io"
	"log"
	"strings"
	"bytes"

	"github.com/PuerkitoBio/goquery"

	"golang.org/x/net/context"
	"golang.org/x/text/language"
	"cloud.google.com/go/translate"
	"google.golang.org/api/option"
	"golang.org/x/net/html"
	"github.com/BurntSushi/toml"
	"os"
	"time"
	"regexp"
)

func main() {
	// scrape_html()
	// translate_html()
	// createClientWithKey()


	urls := [] string {
		"",
	}

	t := time.Now()
	s := t.Format("20060102-150405")
	filename := "/Users/sheroz/Documents/tmp/scraping-" + s + ".txt"
	fmt.Println("Opening file:" + filename)


	f, err := os.Create(filename)
	if err != nil {
		log.Fatal(err)
	}

	defer f.Close()
	totalCount := 0
	for _,url := range urls {

		document, err := goquery.NewDocument(url)
		if err != nil {
			log.Fatal(err)
		}
		var selector string
		var html string
		var text string
		var textSum string

		textSum = ""


		selector = "#product_tabs_description_contents"
		html, text, err = scrape_html(document, selector)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Println(text)
		textSum += text


		selector = "#product_tabs_quickinfo_contents"
		html, text, err = scrape_html(document, selector)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(text)
		textSum += text

		selector = "#product_tabs_additional_contents"
		html, text, err = scrape_html(document, selector)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(text)
		textSum += text

		wordCount := get_word_counts(textSum)

		f.WriteString("***\n")
		f.WriteString(url)
		f.WriteString("\n")
		f.WriteString(textSum)
		f.WriteString("\n")
		f.WriteString(fmt.Sprintf("%v", wordCount))
		f.WriteString("\n")
		f.WriteString("***\n")
		totalCount += wordCount

		_ = html
	}
	f.WriteString("\n")
	f.WriteString("====================\n")
	f.WriteString("Total word count: ")
	f.WriteString(fmt.Sprintf("%v", totalCount))
	f.WriteString("\n")
	f.WriteString("***\n")
	f.Sync()

	fmt.Println("Result saved in:" + filename)

}

func get_word_counts(text string) int {
	words:= regexp.MustCompile("\\b[^\\s]+\\b")
	res := words.FindAllString(text, -1)
	return len(res)
}

func scrape_html(document *goquery.Document, selector string) (string, string, error){

	allowedTags := [] string {
		"h1",
		"h2",
		"h3",
		"h4",
		"h5",
		"h6",
		"a",
		"p",
		"ul",
		"li",
		"img",
		}
	allowedTagsMap := map[string]bool{}
	for _,v := range allowedTags {
		allowedTagsMap[v] = true
	}

	var htmlBuf bytes.Buffer
	var textBuf bytes.Buffer

	// Find the review items
	document.Find(selector).Each(func(i int, s *goquery.Selection) {
		// For each item found, get the band and title
		/*
		band := s.Find("a").Text()
		title := s.Find("i").Text()
		fmt.Printf("Review %d: %s - %s\n", i, band, title)
		*/

		s.Find("script").Remove()
		s.Find("style").Remove()
		s.Find("pre").Remove()
		s.Find("br").Remove()
		s.Find("hr").Remove()

		// count:=0

		var f func(*html.Node)
		f = func(n *html.Node) {
			if n == nil { return }
			//count++
			if n.Type == html.ElementNode {
				_,tagAllowed := allowedTagsMap[n.Data]
				if tagAllowed {
					htmlBuf.WriteString("<")
					text := strings.TrimSpace(n.Data)
					htmlBuf.WriteString(text)
					if n.Data == "a" {
						m := make(map[string]string)
						for _, a := range n.Attr {
							m[a.Key] = a.Val
						}
						href, ok := m["href"]
						if ok {
							htmlBuf.WriteString(" href=\"")
							htmlBuf.WriteString(href)
							htmlBuf.WriteString("\"")
						}
					}
					htmlBuf.WriteString(">")
				}
				f(n.FirstChild)
				if tagAllowed {
					htmlBuf.WriteString("</")
					text := strings.TrimSpace(n.Data)
					htmlBuf.WriteString(text)
					htmlBuf.WriteString(">")
				}
			}
			if n.Type == html.TextNode {
				text := strings.TrimSpace(n.Data)
				htmlBuf.WriteString(text)
			}
			f(n.NextSibling)
			/*
			if n.FirstChild != nil {
				for c := n.FirstChild; c != nil; c = c.NextSibling {
					f(c)
				}
			}
			*/
		}
		for _, n := range s.Nodes {
			f(n)
		}

		// fmt.Println(count)
		textBuf.WriteString(s.Text())

		/*
		for _, n := range s.Nodes {
			n.RemoveAttr("class")
		}
		*/
		// htmlResult, _ = q.Html()
	})

	htmlResult := htmlBuf.String()
	textResult := textBuf.String()

	// fmt.Println("*** HTML result ***")
	// fmt.Println(htmlResult)

	// remove empty lines
	textResult = strings.TrimSpace(textResult)
	startLen := len(textResult)
	for {
		textResult = strings.Replace(textResult, "\n\n", "\n", -1)
		currentLen := len(textResult)
		if startLen == currentLen {
			break
		}
		startLen = currentLen
	}

	/*
	fmt.Println("*** Text result trimmed ***")
	regex, err := regexp.Compile("\n\n")
	if err != nil {
		return
	}
	textResult = regex.ReplaceAllString(textResult, "\n")
	fmt.Println(textResult)
	*/
	return htmlResult, textResult, nil
}

func translate_html() {

	type translationConfig struct {
		Vendor string
		ApiKey string
	}

	type appConfig struct {
		Version string
		Translation translationConfig
	}

	var conf appConfig
	configFile := "config.toml"
	if _, err := toml.DecodeFile(configFile, &conf); err != nil {
		log.Fatal("Failed to decode config file: %v\n", err.Error())
	}

	fmt.Printf("Version: %s\n", conf.Version)
	fmt.Printf("Translation vendor: %s\n", conf.Translation.Vendor)
	// fmt.Printf("Translation apiKey: %s\n", conf.Translation.ApiKey)

	ctx := context.Background()

	client, err := translate.NewClient(ctx, option.WithAPIKey(conf.Translation.ApiKey))
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	// Sets the text to translate.
	text := "<p>Hello, world!</p> <ul>   <li>1</li>    <li>2</li>    </ul>"

	sourceLanguage, err := language.Parse("en")
	if err != nil {
		log.Fatalf("Failed to parse source language: %v", err)
	}

	// Sets the target language.
	targetLanguage, err := language.Parse("ru")
	if err != nil {
		log.Fatalf("Failed to parse target language: %v", err)
	}


	fmt.Print("\n")
	fmt.Printf("TargetLanguage: %s\n", targetLanguage)

	/*
	langs, err := client.SupportedLanguages(ctx, sourceLanguage)
	if err != nil {
		log.Fatalf("Failed to parse supported languages: %v", err)
	}
	for _, lang := range langs {
		fmt.Printf("%q: %s\n", lang.Tag, lang.Name)
	}
	*/

	opts := translate.Options{
		Source:  sourceLanguage,
		Format: "html",
	}

	// Translates the text into Russian.
	translations, err := client.Translate(ctx, []string{text}, targetLanguage, &opts)

	if err != nil {
		log.Fatalf("Failed to translate text: %v", err)
	}

	fmt.Printf("Text: %v\n", text)
	fmt.Printf("Translation: %v\n", translations[0].Text)

}

func createClientWithKey() {
	// import "cloud.google.com/go/translate"
	// import "google.golang.org/api/option"
	// import "golang.org/x/text/language"
	ctx := context.Background()
	const apiKey = ""
	client, err := translate.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		log.Fatal(err)
	}

	resp, err := client.Translate(ctx, []string{"Hello, world!"}, language.Russian, nil)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("%#v", resp)
}

func translateText(targetLanguage, text string) (string, error) {
	ctx := context.Background()

	lang, err := language.Parse(targetLanguage)
	if err != nil {
		return "", err
	}

	client, err := translate.NewClient(ctx)
	if err != nil {
		return "", err
	}
	defer client.Close()

	resp, err := client.Translate(ctx, []string{text}, lang, nil)
	if err != nil {
		return "", err
	}
	return resp[0].Text, nil
}

func detectLanguage(text string) (*translate.Detection, error) {
	ctx := context.Background()
	client, err := translate.NewClient(ctx)
	if err != nil {
		return nil, err
	}
	defer client.Close()

	lang, err := client.DetectLanguage(ctx, []string{text})
	if err != nil {
		return nil, err
	}
	return &lang[0][0], nil
}

func listSupportedLanguages(w io.Writer, targetLanguage string) error {
	ctx := context.Background()

	lang, err := language.Parse(targetLanguage)
	if err != nil {
		return err
	}

	client, err := translate.NewClient(ctx)
	if err != nil {
		return err
	}
	defer client.Close()

	langs, err := client.SupportedLanguages(ctx, lang)
	if err != nil {
		return err
	}

	for _, lang := range langs {
		fmt.Fprintf(w, "%q: %s\n", lang.Tag, lang.Name)
	}

	return nil
}

func translateTextWithModel(targetLanguage, text, model string) (string, error) {
	ctx := context.Background()

	lang, err := language.Parse(targetLanguage)
	if err != nil {
		return "", err
	}

	client, err := translate.NewClient(ctx)
	if err != nil {
		return "", err
	}
	defer client.Close()

	resp, err := client.Translate(ctx, []string{text}, lang, &translate.Options{
		Model: model, // Either "mnt" or "base".
	})
	if err != nil {
		return "", err
	}
	return resp[0].Text, nil
}

func sample_main() {
	ctx := context.Background()

	// Creates a client.
	client, err := translate.NewClient(ctx)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	// Sets the text to translate.
	text := "Hello, world!"
	// Sets the target language.
	target, err := language.Parse("ru")
	if err != nil {
		log.Fatalf("Failed to parse target language: %v", err)
	}

	// Translates the text into Russian.
	translations, err := client.Translate(ctx, []string{text}, target, nil)
	if err != nil {
		log.Fatalf("Failed to translate text: %v", err)
	}

	fmt.Printf("Text: %v\n", text)
	fmt.Printf("Translation: %v\n", translations[0].Text)
}
