package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/chromedp/cdproto/browser"
	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/chromedp"
	"github.com/dadosjusbr/status"
)

type crawler struct {
	downloadTimeout   time.Duration
	collectionTimeout time.Duration
	timeBetweenSteps  time.Duration
	year              string
	month             string
	output            string
}

const (
	contrachequeXPATH = `//*[@id="menu-interno-estatico"]/li[1]/a`
	indenizacoesXPATH = `//*[@id="menu-interno-estatico"]/li[8]/a`
)

func (c crawler) crawl() ([]string, error) {
	// Chromedp setup.
	log.SetOutput(os.Stderr) // Enviando logs para o stderr para não afetar a execução do coletor.
	alloc, allocCancel := chromedp.NewExecAllocator(
		context.Background(),
		append(chromedp.DefaultExecAllocatorOptions[:],
			chromedp.UserAgent("Mozilla/5.0 (Macintosh; Intel Mac OS X 10_14_5) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/77.0.3830.0 Safari/537.36"),
			chromedp.Flag("headless", true), // mude para false para executar com navegador visível.
			chromedp.NoSandbox,
			chromedp.DisableGPU,
		)...,
	)
	defer allocCancel()

	ctx, cancel := chromedp.NewContext(
		alloc,
		chromedp.WithLogf(log.Printf), // remover comentário para depurar
	)
	defer cancel()

	ctx, cancel = context.WithTimeout(ctx, c.collectionTimeout)
	defer cancel()

	// NOTA IMPORTANTE: os prefixos dos nomes dos arquivos tem que ser igual
	// ao esperado no parser MPMS.

	// Contracheque
	log.Printf("Clicando em contracheque(%s/%s)...", c.month, c.year)
	if err := c.navegacaoSite(ctx, contrachequeXPATH); err != nil {
		status.ExitFromError(err)
	}
	log.Printf("Clicado com sucesso!\n")

	log.Printf("Realizando seleção (%s/%s)...", c.month, c.year)
	if err := c.selecionaAno(ctx); err != nil {
		status.ExitFromError(err)
	}
	log.Printf("Seleção realizada com sucesso!\n")
	cqFname := c.downloadFilePath("contracheque")
	log.Printf("Fazendo download do contracheque (%s)...", cqFname)

	if err := c.exportaPlanilha(ctx, cqFname); err != nil {
		status.ExitFromError(err)
	}
	log.Printf("Download realizado com sucesso!\n")

	// Indenizações
	month, _ := strconv.Atoi(c.month)
	if c.year != "2018" || (c.year == "2019" && month >= 6) {
		log.Printf("\nClicando na aba indenizações (%s/%s)...", c.month, c.year)
		if err := c.navegacaoSite(ctx, indenizacoesXPATH); err != nil {
			status.ExitFromError(err)
		}

		log.Printf("Realizando seleção (%s/%s)...", c.month, c.year)
		if err := c.selecionaAno(ctx); err != nil {
			status.ExitFromError(err)
		}

		log.Printf("Seleção realizada com sucesso!\n")
		iFname := c.downloadFilePath("verbas-indenizatorias")
		log.Printf("Fazendo download das indenizações (%s)...", iFname)
		if err := c.exportaPlanilha(ctx, iFname); err != nil {
			status.ExitFromError(err)
		}
		log.Printf("Download realizado com sucesso!\n")

		// Retorna caminhos completos dos arquivos baixados.
		return []string{cqFname, iFname}, nil
	}

	return []string{cqFname}, nil
}

func (c crawler) downloadFilePath(prefix string) string {
	return filepath.Join(c.output, fmt.Sprintf("membros-ativos-%s-%s-%s.xlsx", prefix, c.month, c.year))
}

// Navega para as planilhas
func (c crawler) navegacaoSite(ctx context.Context, xpath string) error {
	const (
		baseURL = "https://transparenciaweb.mpms.mp.br/contracheque"
	)

	if err := chromedp.Run(ctx,
		chromedp.Navigate(baseURL),
		chromedp.Sleep(c.timeBetweenSteps),

		// Abre o contracheque
		chromedp.Click(xpath, chromedp.BySearch, chromedp.NodeVisible),
		chromedp.Sleep(c.timeBetweenSteps),

		// Altera o diretório de download
		browser.SetDownloadBehavior(browser.SetDownloadBehaviorBehaviorAllowAndName).
			WithDownloadPath(c.output).
			WithEventsEnabled(true),
	); err != nil {
		return status.NewError(status.ConnectionError, err)
	}
	return nil
}

func (c crawler) selecionaAno(ctx context.Context) error {

	selectYear := `//*[@id="box-seach"]/form/select`
	//Faz a seleção apenas do ano
	if err := chromedp.Run(ctx,
		// Seleciona ano
		chromedp.SetValue(selectYear, c.year, chromedp.BySearch, chromedp.NodeVisible),
		chromedp.Sleep(c.timeBetweenSteps),

		chromedp.Click(`//*[@id="box-seach"]/form/button`, chromedp.BySearch, chromedp.NodeVisible),
		chromedp.Sleep(c.timeBetweenSteps),
	); err != nil {
		return status.NewError(status.DataUnavailable, err)
	}
	return nil
}

// exportaPlanilha clica no botão correto para exportar para excel, espera um tempo para download e renomeia o arquivo.
func (c crawler) exportaPlanilha(ctx context.Context, fName string) error {
	tctx, tcancel := context.WithTimeout(ctx, 30*time.Second)
	defer tcancel()
	monthConverted, err := strconv.Atoi(c.month)
	if err != nil {
		return fmt.Errorf("erro ao converter mês para inteiro:%w", err)
	}
	var rows []*cdp.Node
	err = chromedp.Run(ctx, chromedp.Nodes("tr", &rows, chromedp.ByQueryAll))
	if err != nil {
		return fmt.Errorf("erro ao selecionar elemento <tr>:%w", err)
	}
	tr := len(rows) + 1 - monthConverted
	link := fmt.Sprintf(`/html/body/div[3]/div[2]/div[3]/div/table/tbody/tr[%d]/td[3]/a`, tr)
	if err := chromedp.Run(tctx,
		// Clica no botão de download
		chromedp.Click(link, chromedp.BySearch, chromedp.NodeVisible),
		chromedp.Sleep(c.timeBetweenSteps),
	); err != nil {
		return status.NewError(status.DataUnavailable, fmt.Errorf("planilha não disponível: %w", err))
	}

	time.Sleep(c.downloadTimeout)

	if err := nomeiaDownload(c.output, fName); err != nil {
		return fmt.Errorf("erro renomeando arquivo (%s): %w", fName, err)
	}
	if _, err := os.Stat(fName); os.IsNotExist(err) {
		return fmt.Errorf("download do arquivo de %s não realizado", fName)
	}
	return nil
}

// nomeiaDownload dá um nome ao último arquivo modificado dentro do diretório
// passado como parâmetro nomeiaDownload dá pega um arquivo
func nomeiaDownload(output, fName string) error {
	// Identifica qual foi o ultimo arquivo
	files, err := os.ReadDir(output)
	if err != nil {
		return fmt.Errorf("erro lendo diretório %s: %w", output, err)
	}
	var newestFPath string
	var newestTime int64 = 0
	for _, f := range files {
		fPath := filepath.Join(output, f.Name())
		fi, err := os.Stat(fPath)
		if err != nil {
			return fmt.Errorf("erro obtendo informações sobre arquivo %s: %w", fPath, err)
		}
		currTime := fi.ModTime().Unix()
		if currTime > newestTime {
			newestTime = currTime
			newestFPath = fPath
		}
	}
	// Renomeia o ultimo arquivo modificado.
	if err := os.Rename(newestFPath, fName); err != nil {
		return fmt.Errorf("erro renomeando último arquivo modificado (%s)->(%s): %w", newestFPath, fName, err)
	}
	return nil
}
