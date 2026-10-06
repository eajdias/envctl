package cli

import (
	"fmt"

	"github.com/pterm/pterm"
	"github.com/pterm/pterm/putils"
)

func PrintBanner() {
	bannerText, err := pterm.DefaultBigText.WithLetters(
		putils.LettersFromStringWithStyle("ENV", pterm.NewStyle(pterm.FgCyan, pterm.Bold)),
		putils.LettersFromStringWithStyle("CTL", pterm.NewStyle(pterm.FgLightMagenta, pterm.Bold)),
	).Srender()
	if err != nil {
		bannerText = "ENVCTL"
	}

	pterm.Println(bannerText)
	pterm.DefaultCenter.Println(pterm.LightCyan("🚀 Universal Environment Provisioner (Windows 11 PRO & Ubuntu Linux)"))
	pterm.DefaultCenter.Println(pterm.Gray("Clean Architecture • Idempotent • Embedded Assets • Multi-OS & Subagent Dispatch\n"))
}

func PrintSection(title string) {
	pterm.DefaultSection.Println(title)
}

func PrintInfo(msg string) {
	pterm.Info.Println(msg)
}

func PrintSecretGuidance() {
	pterm.DefaultBox.WithTitle(pterm.LightYellow("🔐 Post-Provisioning Security & Secrets Guidance")).Println(
		fmt.Sprintf("%s\n%s\n%s\n%s\n%s",
			pterm.Bold.Sprint("1. SSH Private Keys:"),
			"   Copy your VPS private keys (.pem / id_rsa) into: "+pterm.Cyan("~/Documents/SSH-keys/"),
			"   (Permissions were automatically restricted to your Windows User ACLs)",
			pterm.Bold.Sprint("2. SSH Manager & Known Hosts:"),
			"   Configurations located in: "+pterm.Cyan("~/.ssh-manager/")+" and "+pterm.Cyan("~/.ssh/"),
		),
	)
}
