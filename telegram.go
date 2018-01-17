package main

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/jaytaylor/html2text"
)

func getTelegramGroupHistory(group string, grace int, dumpFlag bool, startMessage int, endMessage int) {
	checkGroupName(group)
	if endMessage != 0 {
		if endMessage <= startMessage {
			fmt.Println("[-] The final message number (-e)  must be >= than start message number (-s)")
			os.Exit(1)
		}
	}

	graceCounter := 0
	if endMessage != 0 {
		fmt.Println("[?] End  message set, grace time will be ignored")
	}
	dumpfile := group + ".dump"
	msgtxt := ""

	messageCounter := startMessage
	readFromTelegramDump(dumpfile, dumpFlag, &messageCounter)
	messageCounter++

	startTime := time.Now()
	fmt.Println("==== [" + startTime.Format(time.RFC3339) + "] Dumping messages for " + group + " ====")
	for {
		messageid := strconv.Itoa(messageCounter)
		body := retriveRequestBody("https://t.me/" + group + "/" + messageid + "?embed=1")
		message := getTelegramMessage(body)
		if message != "" && graceCounter > 0 {
			for j := 0; j < graceCounter; j++ {
				msg := "[MESSAGE REMOVED]"
				if dumpFlag {
					writeOnFile(dumpfile, "["+strconv.Itoa(messageCounter)+"] "+msg+"\n")
				}
				fmt.Println(msg)
			}
			graceCounter = 0
		} else if message != "" {
			username, nickname := getTelegramUsername(body)
			date, time := getTelegramMessageDateTime(body)

			if username == "" {
				//for channels
				if nickname == "" {
					msgtxt = "[" + date + " " + time + "] " + message
				} else {
					msgtxt = "[" + date + " " + time + "] " + nickname + ": " + message
				}
			} else {
				msgtxt = "[" + date + " " + time + "] " + nickname + "(" + username + "): " + message
			}

			msg, _ := html2text.FromString(msgtxt)
			if dumpFlag {
				writeOnFile(dumpfile, "["+messageid+"] "+strings.Replace(msg, "\n", " ⏎ ", -1)+"\n")
			}
			fmt.Println(msg)
		} else {
			if messageCounter == 1 {
				fmt.Println("[!!] Invalid group")
				break
			}
			if endMessage == 0 {
				graceCounter++
				if graceCounter == grace {
					messageCounter = messageCounter - graceCounter
					break
				}
			} else {
				msg := "[DELETED MESSAGE]"
				if dumpFlag {
					writeOnFile(dumpfile, "["+strconv.Itoa(messageCounter)+"] "+msg+"\n")
				}
				fmt.Println(msg)
			}
		}
		if endMessage != 0 {
			if messageCounter == endMessage {
				messageCounter--
				break
			}
		}
		messageCounter++
		time.Sleep(time.Millisecond * 100)
	}
	fmt.Println("==== [" + time.Now().Format(time.RFC3339) + " (elapsed:" + time.Since(startTime).String() + ")] End of history - " + strconv.Itoa(messageCounter-startMessage) + " messages scraped  ==== ")
	if endMessage == 0 && messageCounter > 0 {
		fmt.Println("[=] If you think there are more messages try to increase the grace period (--grace [INT])")
	}
}

func getTelegramMessage(body string) string {
	re := regexp.MustCompile(`class=\"tgme_widget_message_text\" dir=\"auto\">(.*)<\/div>\n`)
	match := re.FindAllStringSubmatch(body, -1)
	messageBody := ""
	if len(match) == 1 {
		messageBody = match[0][1]
	} else if len(match) == 2 {
		quotedUser := getMessageRepliedAuthor(body)
		messageBody = "{ " + quotedUser + match[0][1] + " } " + match[1][1]
	}
	messageBody = messageBody + getTelegramMedia(body)
	return messageBody
}

func getTelegramMedia(body string) string {
	messageBody := getTelegramVideo(body) + getTelegramPhoto(body) + getTelegramVoice(body) + getTelegramServiceMessage(body) + getTelegramDocument(body)
	return messageBody
}

func getTelegramPhoto(body string) string {
	re := regexp.MustCompile(`image:url\('(https:\/\/[\w+.\/-]+)'`)
	match := re.FindStringSubmatch(body)
	if len(match) == 2 {
		return "Image: " + match[1]
	}
	return ""
}

func getTelegramVoice(body string) string {
	re := regexp.MustCompile(`voice"\ssrc="(https:\/\/[\w.\/-]+)"`)
	match := re.FindStringSubmatch(body)
	if len(match) == 2 {
		return "Voice: " + match[1]
	}
	return ""
}

func getTelegramVideo(body string) string {
	re := regexp.MustCompile(`video"?\ssrc="(https:\/\/[\w.\/-]+)"`)
	match := re.FindStringSubmatch(body)
	if len(match) == 2 {
		return "Video: " + match[1]
	}
	return ""
}

func getTelegramDocument(body string) string {
	re := regexp.MustCompile(`document_title" dir="auto">(.*)</div>`)
	match := re.FindStringSubmatch(body)
	if len(match) == 2 {
		return "Document: " + match[1]
	}
	return ""
}

func getTelegramServiceMessage(body string) string {
	re := regexp.MustCompile(`<div\sclass="message_media_not_supported_label">Service\smessage<\/div>`)
	if re.MatchString(body) {
		return "[SERVICE MESSAGE]"
	}
	return ""
}

func getMessageRepliedAuthor(body string) string {
	re := regexp.MustCompile(`reply"\shref="https:\/\/t.me\/[\w/]+">[\n\s]+<div\sclass="tgme_widget_message_author">[\n\s]+<span\sclass="tgme_widget_message_author_name"\s?dir="auto">(.*)</span>`)
	match := re.FindStringSubmatch(body)
	if len(match) == 2 {
		return " " + match[1] + ": "
	}
	return ""
}

func getTelegramUsername(body string) (string, string) {
	re := regexp.MustCompile(`class=\"tgme_widget_message_author_name\"\s?(?:href="https://t\.me/(\w+)")? dir=\"auto\">(.*)<\/(?:span>)?(?:a>)?&nbsp;in&nbsp;<a`)
	match := re.FindStringSubmatch(body)
	if len(match) == 3 {
		return match[1], match[2]
	} else {
		return "", ""
	}
}

func getTelegramMessageDateTime(body string) (string, string) {
	re := regexp.MustCompile(`<time datetime="(\d+-\d+-\d+)T(\d+:\d+:\d+)\+\d+:\d+\">`)
	match := re.FindStringSubmatch(body)
	return match[1], match[2]
}

func checkGroupName(group string) {
	re := regexp.MustCompile(`^[[:alpha:]](?:\-?[[:alnum:]]){3,31}$`)
	if !re.MatchString(group) {
		fmt.Println("Invalid Group name, valid chars alphanum, -")
		os.Exit(1)
	}
}

func readFromTelegramDump(dumpfile string, dumpFlag bool, messageCounter *int) {
	if dumpFlag {
		fmt.Println("[=] --dumpfile used, ignoring --startpoint")
		if fileExists(dumpfile) {
			fmt.Println("[=] The dump will be saved in " + dumpfile)
			fmt.Println("[?] Print the existing dumb before resuming it? [Y/N]")
			var resp string
			_, err := fmt.Scanln(&resp)
			if err != nil {
				fmt.Println("[-] Unable to read answer")
				os.Exit(1)
			}
			fmt.Println("[+] Calculating the last message")
			file, _ := os.Open(dumpfile)
			scan := bufio.NewScanner(file)
			for scan.Scan() {
				messageSlice := strings.Split(scan.Text(), " ")
				if resp == "y" || resp == "Y" {
					fmt.Println(strings.Join(messageSlice[1:], " "))
				}
				*messageCounter, _ = strconv.Atoi(strings.Trim(messageSlice[0], "[]"))
			}
			fmt.Println("[=] Starting from message n.", *messageCounter)
		}
	}
}
