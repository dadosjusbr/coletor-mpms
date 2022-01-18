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
	"github.com/chromedp/chromedp"
)

type crawler struct {
	collectionTimeout time.Duration
	timeBetweenSteps  time.Duration
	year              string
	month             string
	output            string
}

const (
	contrachequeXPATH = "//*[@id='20']"
	indenizacoesXPATH = "//*[@id='83']/div[2]/table/tbody"
	// indenizacoesXPATH = "//*[@id='59']"
)

func (c crawler) crawl() ([]string, error) {
	// Chromedp setup.
	log.SetOutput(os.Stderr) // Enviando logs para o stderr para não afetar a execução do coletor.
	alloc, allocCancel := chromedp.NewExecAllocator(
		context.Background(),
		append(chromedp.DefaultExecAllocatorOptions[:],
			chromedp.UserAgent("Mozilla/5.0 (Macintosh; Intel Mac OS X 10_14_5) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/77.0.3830.0 Safari/537.36"),
			chromedp.Flag("headless", false), // mude para false para executar com navegador visível.
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

	// Seleciona o contracheque na página principal
	log.Printf("Clicando em contracheque(%s/%s)...", c.month, c.year)
	if err := c.navegacaoSite(ctx, contrachequeXPATH); err != nil {
		log.Fatalf("Erro no setup:%v", err)
	}
	log.Printf("Clicado com sucesso!\n")

	// Contracheque
	log.Printf("Realizando seleção (%s/%s)...", c.month, c.year)
	if err := c.selecionaMesAno(ctx); err != nil {
		log.Fatalf("Erro no setup:%v", err)
	}
	log.Printf("Seleção realizada com sucesso!\n")
	cqFname := c.downloadFilePath("contracheque")
	log.Printf("Fazendo download do contracheque (%s)...", cqFname)

	if err := c.exportaPlanilha(ctx, cqFname); err != nil {
		log.Fatalf("Erro fazendo download do contracheque: %v", err)
	}
	log.Printf("Download realizado com sucesso!\n")

	// Indenizações
	month, _ := strconv.Atoi(c.month)
	if c.year == "2020" || c.year == "2021" || (c.year == "2019" && month >= 7) {
		log.Printf("\nClicando na aba indenizações (%s/%s)...", c.month, c.year)
		if err := c.clicaAba(ctx, indenizacoesXPATH); err != nil {
			log.Fatalf("Erro no setup:%v", err)
		}

		log.Printf("Realizando seleção (%s/%s)...", c.month, c.year)
		if err := c.selecionaMesAno(ctx); err != nil {
			log.Fatalf("Erro no setup:%v", err)
		}

		// log.Printf("Seleção realizada com sucesso!\n")
		iFname := c.downloadFilePath("verbas-indenizatorias")
		log.Printf("Fazendo download das indenizações (%s)...", iFname)
		if err := c.exportaPlanilha(ctx, iFname); err != nil {
			log.Fatalf("Erro fazendo download dos indenizações: %v", err)
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
		baseURL = "https://transparencia.mpms.mp.br/QvAJAXZfc/opendoc.htm?document=portaltransparencia%5Cportaltransparencia.qvw&lang=pt-BR&host=QVS%40srv-1645&anonymous=true"
	)

	return chromedp.Run(ctx,
		chromedp.Navigate(baseURL),
		chromedp.Sleep(c.timeBetweenSteps),

		// Abre o contracheque
		chromedp.Click(xpath, chromedp.BySearch, chromedp.NodeReady),
		chromedp.Sleep(c.timeBetweenSteps),
		
		
		chromedp.Click(`//*[@id="26"]/div[2]/table`, chromedp.BySearch, chromedp.NodeReady),
		chromedp.Sleep(c.timeBetweenSteps),

		// Altera o diretório de download
		browser.SetDownloadBehavior(browser.SetDownloadBehaviorBehaviorAllowAndName).
			WithDownloadPath(c.output).
			WithEventsEnabled(true),
	)
}

// clicaAba clica na aba referenciada pelo XPATH passado como parâmetro.
// Também espera até o estar visível.
func (c crawler) clicaAba(ctx context.Context, xpath string) error {
	return chromedp.Run(ctx,
		chromedp.Click(xpath),
		chromedp.Sleep(c.timeBetweenSteps),
	)
}

func (c crawler) selecionaMesAno(ctx context.Context) error {
	if c.month == "01" {
		c.month = "Jan"
	} else if c.month == "02" {
		c.month = "Fev"
	} else if c.month == "03" {
		c.month = "Mar"
	} else if c.month == "04" {
		c.month = "Abr"
	} else if c.month == "05" {
		c.month = "Mai"
	} else if c.month == "06" {
		c.month = "Jun"
	} else if c.month == "07" {
		c.month = "Jul"
	} else if c.month == "08" {
		c.month = "Ago"
	} else if c.month == "09" {
		c.month = "Set"
	} else if c.month == "10" {
		c.month = "Out"
	} else if c.month == "11" {
		c.month = "Nov"
	} else if c.month == "12" {
		c.month = "Dez"
	}

	selectYear := "//*[@title='Ano']/div"
	selectMonth := "//*[@title='Mês']/div"

	year := fmt.Sprintf("//*[@title='%s']", c.year)
	month := fmt.Sprintf("//*[@title='%s']", c.month)


	if c.year == "2021"{
		if c.month== "Dez"{
			return chromedp.Run(ctx)
		}

		return chromedp.Run(ctx,
			// Espera ficar visível
			chromedp.WaitVisible(selectYear, chromedp.BySearch, chromedp.NodeReady),
	
			// Seleciona mês
			chromedp.Click(selectMonth, chromedp.BySearch, chromedp.NodeReady),
			chromedp.Sleep(c.timeBetweenSteps),
	
			chromedp.Click(month, chromedp.BySearch, chromedp.NodeReady),
			chromedp.Sleep(c.timeBetweenSteps),
		)
	}

	if c.month == "Dez"{
		return chromedp.Run(ctx,
			// Espera ficar visível
			chromedp.WaitVisible(selectYear, chromedp.BySearch, chromedp.NodeReady),
	
			// Seleciona ano
			chromedp.Click(selectYear, chromedp.BySearch, chromedp.NodeReady),
			chromedp.Sleep(c.timeBetweenSteps),
	
			chromedp.Click(year, chromedp.BySearch, chromedp.NodeReady),
			chromedp.Sleep(c.timeBetweenSteps),
		)

	}

	return chromedp.Run(ctx,
		// Espera ficar visível
		chromedp.WaitVisible(selectYear, chromedp.BySearch, chromedp.NodeReady),

		// Seleciona ano
		chromedp.Click(selectYear, chromedp.BySearch, chromedp.NodeReady),
		chromedp.Sleep(c.timeBetweenSteps),

		chromedp.Click(year, chromedp.BySearch, chromedp.NodeReady),
		chromedp.Sleep(c.timeBetweenSteps),

		// Seleciona mês
		chromedp.Click(selectMonth, chromedp.BySearch, chromedp.NodeReady),
		chromedp.Sleep(c.timeBetweenSteps),

		chromedp.Click(month, chromedp.BySearch, chromedp.NodeReady),
		chromedp.Sleep(c.timeBetweenSteps),
	)
		
}

// exportaPlanilha clica no botão correto para exportar para excel, espera um tempo para download renomeia o arquivo.
func (c crawler) exportaPlanilha(ctx context.Context, fName string) error {
	chromedp.Run(ctx,
		// Clica no botão de download
		chromedp.Click(`//*[@title='Enviar para Excel']`, chromedp.BySearch, chromedp.NodeVisible),
		chromedp.Sleep(c.timeBetweenSteps),
	)

	if err := nomeiaDownload(c.output, fName); err != nil {
		return fmt.Errorf("erro renomeando arquivo (%s): %v", fName, err)
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
		return fmt.Errorf("erro lendo diretório %s: %v", output, err)
	}
	var newestFPath string
	var newestTime int64 = 0
	for _, f := range files {
		fPath := filepath.Join(output, f.Name())
		fi, err := os.Stat(fPath)
		if err != nil {
			return fmt.Errorf("erro obtendo informações sobre arquivo %s: %v", fPath, err)
		}
		currTime := fi.ModTime().Unix()
		if currTime > newestTime {
			newestTime = currTime
			newestFPath = fPath
		}
	}
	// Renomeia o ultimo arquivo modificado.
	if err := os.Rename(newestFPath, fName); err != nil {
		return fmt.Errorf("erro renomeando último arquivo modificado (%s)->(%s): %v", newestFPath, fName, err)
	}
	return nil
}
