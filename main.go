package main

import (
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"log"
	"net/http"
	_ "novel/conf"
	"novel/entity"
	"novel/serve"
	"novel/util"
	"strconv"
	"strings"
	"time"
)

var (
	booker = new(serve.BookServe)
)

// 1. 每天00：00同步小说书籍类型，作者，书籍
// 2. 每天12：00；17：00；00：00更新小说节章，
func main() {

	//同步作者，书记籍类型
	//Type()

	//同步书籍
	Books()

}

// Books 通过作者获取书籍
func Books() {

	url := "https://www.znlzd.com/ar.php?keyWord="
	//获取所有的作者信息
	authors := booker.FindByAuthors()
	for _, author := range authors {

		body := HttpGet(fmt.Sprint(url + author.Author))
		dom := DomInfo(body)
		html := dom.Find("ul[class='txt-list txt-list-row5']")
		selection := html.Find("li")
		//获取到每个作品名称 key  value：url
		BookUrl := GetFindName(selection)
		ForUrl(BookUrl)

		time.Sleep(time.Second * 10)
	}

}

// ForUrl 获取切片内的 url
func ForUrl(url []string) {
	for _, val := range url {
		pageNum := 1
		for true {
			body := HttpGet(fmt.Sprint("https://www.znlzd.com" + val + "index_" + strconv.Itoa(pageNum) + ".html#main"))
			logrus.Println("当前地址--", fmt.Sprint("https://www.znlzd.com"+val+"index_"+strconv.Itoa(pageNum)+".html#main"))
			dom := DomInfo(body)
			html := dom.Find("div[class='info']")

			//更新书籍状态
			BookDomInfo(html)

			//更新封面
			imgbox := dom.Find("div[class='imgbox']")
			ImgBox(imgbox)

			////获取正文 节章目录
			SectionHtml := dom.Find("div[class='layout layout-col1']")
			BookUrl := FindSection(SectionHtml)

			ForContentUrl(BookUrl)

			//获取分页
			pageSelect := dom.Find("div[class='listpage']")
			pageSelectTrue := PageSelect(pageSelect)
			if !pageSelectTrue {
				break
			}
			pageNum++
			time.Sleep(time.Second * 10)
		}
	}
}

// ForContentUrl 获取节章数据
func ForContentUrl(url []string) {
	for _, val := range url {

		body := HttpGet(fmt.Sprint("https://www.znlzd.com" + val))
		logrus.Println("节章内容路径-----", fmt.Sprint("https://www.znlzd.com"+val))

		//创建  Document 对象
		dom := DomInfo(body)

		DomContent(dom)

		time.Sleep(time.Second * 10)
	}
}

func DomContent(dom *goquery.Document) {

	title1 := dom.Find("h1[class='title']").Text()
	content1, _ := dom.Find("div[class='content']").Html()

	content := content1[strings.Index(content1, "</div>")+6:strings.LastIndex(content1, "<br/>")] + "<br/>"

	selection := dom.Find("div[class='section-opt']")

	var strTrue bool
	var url string

	selection.Each(func(i int, sel *goquery.Selection) {
		text, _ := sel.Find("a").Eq(2).Html()
		if text == "下一页" {
			href, exits := sel.Find("a").Eq(2).Attr("href")
			if exits {
				url = href
			}
			strTrue = true
		} else {
			strTrue = false
		}
	})

	if strTrue {
		bodys := HttpGet(fmt.Sprint("https://www.znlzd.com" + url[:strings.LastIndex(url, ".")] + "_2" + url[strings.LastIndex(url, "."):]))
		logrus.Println("节章下一页内容路径-----", fmt.Sprint("https://www.znlzd.com"+url[:strings.LastIndex(url, ".")]+"_2"+url[strings.LastIndex(url, "."):]))
		dom2 := DomInfo(bodys)
		title2 := dom2.Find("h1[class='title']").Text()
		content2, _ := dom2.Find("div[class='content']").Html()
		if title2 == title1 {
			content = content + content2[strings.Index(content2, "</div>")+6:strings.LastIndex(content2, "<br/>")] + "<br/>"
		}
	}

	//存储数据
	var Content entity.Content
	Content.Content = content
	Content.Section = title1

	err := booker.UpdateContent(Content)
	if err != nil {
		logrus.Println("content 内容更新失败---", err)
	} else {
		logrus.Println("content 内容更新成功---")
		logrus.Println("书籍名称---", Content.Section)
	}

}

func PageSelect(info *goquery.Selection) bool {

	_, exit := info.Find("span[class='right']").Find("a").Attr("href")

	return exit
}

func FindSection(info *goquery.Selection) (BookUrl []string) {

	h2 := strings.TrimSpace(info.Find("h2[class='layout-tit']").Last().Text())
	div := info.Find("ul[class='section-list fix']").Last()

	div.Each(func(i int, s *goquery.Selection) {
		//书籍名称
		var contentSection []string
		s.Find("li").Each(func(i int, Se *goquery.Selection) {
			section := Se.Find("a").Text()
			contentSection = append(contentSection, strings.TrimSpace(section))
		})

		s.Find("li").Each(func(i int, s *goquery.Selection) {
			attr, exists := s.Find("a").Attr("href")
			if exists {
				BookUrl = append(BookUrl, strings.TrimSpace(attr))
			}
		})

		//处理 h2
		h2 = h2[strings.Index(h2, "《")+3 : strings.LastIndex(h2, "》")] //《我投篮实在太准了

		//查询书籍名称是否存在
		booksName := booker.FindBooksNameById(h2)
		if (booksName != entity.Books{}) {

			var Content entity.Content

			for _, val := range contentSection {
				Content.CreateTime = util.TimeNow()
				Content.Status = 1
				Content.BooksId = booksName.Id
				Content.Section = val

				count := booker.FindContentByCount(Content)
				if count == 0 {
					err := booker.CreateContent(Content)
					if err != nil {
						logrus.Println("content--节章插入失败", err)
					} else {
						logrus.Println("content--节章插入成功")
						logrus.Println("节章目录--", Content.Section)
					}

				}
			}
		}
	})
	return
}

func ImgBox(info *goquery.Selection) {
	alt, exists := info.Find("img").Attr("alt")
	src, exists := info.Find("img").Attr("src")

	if exists {
		var book entity.Books
		book.Name = alt
		book.Url = src

		err := booker.UpdateBooksUrl(book)

		if err != nil {
			logrus.Println("books 图片更新失败", err)
		} else {
			logrus.Println("books 图片更新成功")
			logrus.Println("书籍名称--", alt)
			logrus.Println("图片路径--", src)
		}
	}

}

//BookDomInfo 处理 info 获取书籍名称 作者 类别 状态  简介
func BookDomInfo(info *goquery.Selection) {
	info.Each(func(i int, s *goquery.Selection) {

		//书籍名称
		title := info.Find("h1").Text()
		//作者
		authors := strings.Split(info.Find("p:first-child").Text(), "：")[1:]
		//类别
		bookTypes := strings.Split(info.Find("p[class='xs-show']").Eq(0).Text(), "：")[1:]
		bookStatuss := strings.Split(info.Find("p[class='xs-show']").Eq(1).Text(), "：")[1:]
		//简介
		bookIntroduction := info.Find("div[class='desc xs-hidden']").Text()

		//获取对象
		var BookType entity.BookType
		var Author entity.Author
		var Books entity.Books

		var (
			bookStatus string
		)

		for _, val := range authors {
			Author = booker.FindAuthorsById(val)
		}
		for _, val := range bookTypes {
			BookType = booker.FindBookTypeById(val)
		}
		for _, val := range bookStatuss {
			bookStatus = val
		}

		if bookStatus == "连载" || bookStatus == "已连载" {
			Books.BookStatus = 1
		} else if bookStatus == "完结" || bookStatus == "已完结" || bookStatus == "已完本" {
			Books.BookStatus = 2
		}

		Books.Name = title
		Books.AuthorId = Author.Id
		Books.BookTypeId = BookType.Id
		Books.Describe = strings.TrimSpace(bookIntroduction)

		//更新数据
		err := booker.UpdateBooks(Books)
		if err != nil {
			logrus.Println("books 数据更新失败", err)
		} else {
			logrus.Println("books 数据更新成功--")
			logrus.Println("书籍名称--", title)
			logrus.Println("书籍状态--", bookStatus)
			logrus.Println("作者--", authors)
			logrus.Println("类别--", bookTypes, bookStatuss)
			logrus.Println("简介--", Books.Describe)
		}
	})
}

//DomInfo 获取 dom 解析书籍 连载状态 简介信息
func DomInfo(body string) (dom *goquery.Document) {
	dom, err := goquery.NewDocumentFromReader(strings.NewReader(body))
	if err != nil {
		log.Fatalln(err) // 输出并退出程序
	}
	return dom
}

// GetFindName 获取作品名称
func GetFindName(lis *goquery.Selection) (BookUrl []string) {
	lis = lis.Each(func(i int, s *goquery.Selection) {
		s.Find("b").Remove() //作品分类
	})

	lis.Each(func(i int, s *goquery.Selection) {

		find1 := s.Find("span[class='s1']").Text() //作品分类
		find2 := s.Find("span[class='s2']").Text() //作品名称
		findHref, exit := s.Find("a").Attr("href")
		find4 := s.Find("span[class='s4']").Text() //作者

		if find1 == "" || find4 == "" {
			return
		}

		var Books entity.Books
		find1 = find1[1 : len(find1)-1]
		//通过作品分类 获取对象
		BookType := booker.FindBookTypeById(find1)
		//通过作者 获取对象
		Authors := booker.FindAuthorsById(find4)

		if Authors == (entity.Author{}) && find4 != "" {
			//创建作者
			var author entity.Author
			author.Author = find4
			author.CreateTime = util.TimeNow()
			author.Status = 1
			//插入
			err := booker.CreateAuthor(author)
			if err != nil {
				logrus.Println("作者插入失败")
			}
			logrus.Println("作者信息保存完毕--", find4)
			Authors = booker.FindAuthorsById(find4)
		}

		Books.Name = strings.TrimSpace(find2)
		Books.Status = 1
		Books.CreateTime = util.TimeNow()
		Books.BookTypeId = BookType.Id
		Books.AuthorId = Authors.Id

		booksCount := booker.FindBooksByName(Books.Name)
		if booksCount == 0 {
			err := booker.CreateBooks(Books)
			if err != nil {
				logrus.Println("books--数据插入失败")
			}
			logrus.Println("作品分类保存完毕--", find1)
			logrus.Println("作品名称保存完毕--", find2)
		} else {
			logrus.Println("书籍数据已经存在--", Books.Name, "作者--", find4)
		}

		if exit {
			BookUrl = append(BookUrl, strings.TrimSpace(findHref))
		}

	})
	return BookUrl
}

// Type 获取类型
func Type() {
	types := []int{1, 2, 3, 4, 5, 6, 7}
	for _, v := range types {
		Page(v)
	}
}

func Page(typ int) {
	var pageNum int64 = 1
	for true {
		url := "https://www.znlzd.com/ksl/" + fmt.Sprint(typ) + "/" + fmt.Sprint(pageNum) + ".html"
		body := HttpGet(url)
		dom := DomInfo(body)
		// 1.元素选择器
		uls := dom.Find("ul[class='txt-list txt-list-row5']")
		GetFind(typ, uls.Children())

		hd := dom.Find("span[class='hd']").Text()

		lastIndex := strings.LastIndex(hd, "/")

		num := hd[lastIndex+1:]
		parseInt, _ := strconv.ParseInt(num, 10, 64)

		if pageNum >= parseInt {
			return
		}
		pageNum++
		time.Sleep(time.Second * 10)
	}
	logrus.Println("爬取作者，书籍类型完毕")
}

func HttpGet(url string) string {
	res, err := http.Get(url)
	if err != nil {
		log.Fatal(err)
	}
	body, err := ioutil.ReadAll(res.Body)
	defer res.Body.Close()
	if err != nil {
		logrus.Fatal("read error", err)
		return ""
	}

	return string(body)

}

func GetFind(typ int, lis *goquery.Selection) {
	lis.Each(func(i int, s *goquery.Selection) {

		find := s.Find("span[class='s1']")  //目录
		find4 := s.Find("span[class='s4']") //作者

		var bookType entity.BookType
		var author entity.Author

		bookType.Name = find.Text()[1 : len(find.Text())-1]
		bookType.CreateTime = util.TimeNow()
		bookType.Sort = typ
		bookType.Status = 1

		author.Author = find4.Text()
		author.CreateTime = util.TimeNow()
		author.Status = 1

		//查询作者是否存在
		countAuthor := booker.FindByAuthor(author.Author)
		if countAuthor != 0 {
			//查询书籍类型名称是否存在
			countBookType := booker.FindByBookType(bookType.Name)
			if countBookType == 0 {
				//插入
				err := booker.CreateBookType(bookType)
				if err != nil {
					logrus.Fatal("书籍类型插入失败")
				}
			}
			return
		}

		//插入
		err := booker.CreateAuthor(author)
		if err != nil {
			logrus.Println("作者插入失败")
		}

		logrus.Println("目录信息保存完毕--", find)
		logrus.Println("作者信息保存完毕--", find4)
	})
}
