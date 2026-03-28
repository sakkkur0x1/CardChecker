package main

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const (
	colorReset = "\033[0m"
	colorBold  = "\033[1m"
	colorPink  = "\033[38;2;255;182;193m"
	colorRed   = "\033[31m"
	colorGreen = "\033[32m"
	colorCyan  = "\033[36m"
	colorGray  = "\033[90m"

	greetingLocal = "\n" + colorPink + colorBold + " ❯❯❯ BIN Checker " + colorReset +
		colorGray + " | " + colorReset + "Enter 6+ digits\n" +
		colorPink + " 💳 Card number: " + colorReset

	wrongCardLocal = colorRed + " ⚠️  Error: Min 6 digits required. Please try again." + colorReset

	bankFoundLocal = colorGreen + colorBold + " [✔] SUCCESS: " + colorReset +
		"Information retrieved for BIN: " + colorCyan + "%s" + colorReset

	bankNotFoundLocal = colorRed + " [✘] NOT FOUND: " + colorReset +
		"No data for this BIN. Check the numbers."
)

func main() {
	for {
		clearConsole()
		fmt.Print(greetingLocal)
		scanner := bufio.NewScanner(os.Stdin)
		if scanner.Scan() {
			card := scanner.Text()
			card = strings.TrimSpace(card)
			card = strings.ReplaceAll(card, " ", "")

			if card == "del" {
				deleteDB()
				continue
			}

			if len(card) >= 6 {
				findBank(card)
				if !askContinue() {
					fmt.Println(colorGray + " Goodbye! ✨" + colorReset)
					return
				}
			} else {
				fmt.Println(wrongCardLocal)
			}
		}
	}
}

func findBank(cardNum string) {
	dbFile := getDatabasePath()
	dbURL := "https://raw.githubusercontent.com/venelinkochev/bin-list-data/refs/heads/master/bin-list-data.csv"

	if _, err := os.Stat(dbFile); os.IsNotExist(err) {
		err := downloadDB(dbURL, dbFile)
		if err != nil {
			fmt.Printf(colorRed+"\n ❌ Failed to download DB: %v\n", err)
			bufio.NewReader(os.Stdin).ReadString('\n')
			return
		}
		fmt.Println(colorGreen + " Done!" + colorReset)
	}
	file, e := os.Open(dbFile)
	if e != nil {
		fmt.Printf(colorRed+"An error occurred while opening file. \n%s", e)
		bufio.NewReader(os.Stdin).ReadString('\n')
		return
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, e := reader.ReadAll()

	if e != nil {
		fmt.Printf(colorRed+"An error occurred while reading file. \n%s", e)
		bufio.NewReader(os.Stdin).ReadString('\n')
		return
	}

	trimCard := cardNum[:6]

	for _, record := range records {
		if strings.Contains(record[0], trimCard) {
			clearConsole()
			card := CardInfo{
				Bin:         record[0],
				Brand:       record[1],
				Type:        record[2],
				Category:    record[3],
				Issuer:      record[4],
				IssuerPhone: record[5],
				IssuerUrl:   record[6],
				IsoCode2:    record[7],
				IsoCode3:    record[8],
				Country:     record[9],
			}
			fmt.Printf(bankFoundLocal, cardNum+"\n")
			card.show()
			return
		}
	}
	fmt.Println(bankNotFoundLocal)
}

type CardInfo struct {
	Bin         string
	Brand       string
	Type        string
	Category    string
	Issuer      string
	IssuerPhone string
	IssuerUrl   string
	IsoCode2    string
	IsoCode3    string
	Country     string
}

func (info CardInfo) show() {
	fmt.Println(colorPink + colorBold + "\n  💳 CARD INFORMATION FOUND:" + colorReset)
	fmt.Println(colorGray + "╭──────────────────────────────────────────╮" + colorReset)
	printRow("BIN", info.Bin, colorPink+colorBold)
	fmt.Println(colorGray + "├──────────────────────────────────────────┤" + colorReset)

	printRow("Brand", info.Brand, colorCyan)
	printRow("Type", info.Type, colorCyan)
	printRow("Category", info.Category, colorCyan)
	printRow("Issuer", info.Issuer, colorCyan)
	printRow("Phone", info.IssuerPhone, colorCyan)
	printRow("Website", info.IssuerUrl, colorCyan)

	location := info.Country
	if info.IsoCode2 != "" {
		location += " (" + info.IsoCode2 + ")"
	}
	printRow("Location", location, colorCyan)

	fmt.Println(colorGray + "╰──────────────────────────────────────────╯" + colorReset)
}

func downloadDB(url, filename string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	out, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer out.Close()

	pt := &PassThru{
		Writer: out,
		Total:  resp.ContentLength,
	}
	_, err = io.Copy(pt, resp.Body)
	fmt.Println()
	return err
}

func deleteDB() {
	dbFile := getDatabasePath()

	if _, err := os.Stat(dbFile); os.IsNotExist(err) {
		fmt.Println(colorRed + " ℹ️  Database file already doesn't exist." + colorReset)
		return
	}

	err := os.Remove(dbFile)
	if err != nil {
		fmt.Printf(colorRed+" ❌ Failed to delete database: %v\n"+colorReset, err)
	} else {
		clearConsole()
		fmt.Println(colorGreen + " 🗑️  Database deleted successfully! It will be re-downloaded on next search." + colorReset)
	}
}

type PassThru struct {
	io.Writer
	Total   int64
	Current int64
}

func (pt *PassThru) Write(p []byte) (int, error) {
	n, err := pt.Writer.Write(p)
	pt.Current += int64(n)

	mb := float64(pt.Current) / 1024 / 1024

	if pt.Total <= 0 {
		fmt.Printf("\r ⏳ Downloading database: %.2f MB... ", mb)
	} else {
		percent := float64(pt.Current) / float64(pt.Total) * 100
		fmt.Printf("\r ⏳ Downloading database: %.2f%% [%.2f MB / %.2f MB]",
			percent, mb, float64(pt.Total)/1024/1024)
	}

	return n, err
}

func askContinue() bool {
	fmt.Print(colorPink + "\n ❯ Do you want to check another card? (y/n): " + colorReset)

	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		answer := strings.ToLower(strings.TrimSpace(scanner.Text()))
		// Проверяем на "y" или "yes". Если пустой ввод или "n" — возвращаем false
		if answer == "y" || answer == "yes" {
			return true
		}
	}
	return false
}

func printRow(label, value, color string) {
	if value == "" || value == "N/A" {
		value = "Not available"
	}

	if len(value) > 25 {
		value = value[:22] + "..."
	}

	fmt.Printf("%s│%s %-12s : %s%-25s %s│\n",
		colorGray, colorReset, label, color, value, colorGray)
}

func getDatabasePath() string {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return "db.csv"
	}

	appCacheDir := filepath.Join(cacheDir, "cardChecker")
	os.MkdirAll(appCacheDir, 0755)

	return filepath.Join(appCacheDir, "db.csv")
}

func clearConsole() {
	fmt.Print("\033[H\033[2J")
}
