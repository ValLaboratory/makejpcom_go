package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/ValLaboratory/go-ekispert/core"
)

// 表示線区関連
const DISPLINE_MST_ID = 100
const DISPLINE_PTN_ID = 101
const DISPLINE_TRN_ID = 105
const DISPLINE_TRN_FILE = "train_disp_line_ptn.dat"

var (
	// カレントディレクトリのDLLを読み込む
	api = core.MustInitializeWithOption(&core.InitializeOption{
		LibraryDirectory: `C:/Program Files (x86)/ExpWin32/`,
	})

	// コマンドライン引数
	knbdir  = flag.String("k", "", "knbフォルダ位置")
	diadir  = flag.String("d", "", "dataフォルダ位置")
	outfile = flag.String("o", "", "jpcom.knb出力ファイル位置")
	version = flag.String("v", "", "バージョン(yyyyMMdd)")
	idtable = flag.String("t", "", "idtable.txtファイル位置")
)

// 鉄道ダイヤデータの初期処理
func initDiaData(diaDir string) core.ExpDiaDBHandler {
	fileList := getEntryFileList(diaDir, ".dat")
	if len(fileList) == 0 {
		return nil
	}

	diadbFileList := api.ExpDiaDB_NewFileList()

	if diadbFileList != nil {
		var ioLevel core.ExpInt = core.EXP_IO_LEVEL_WIN32_AND_MMAP

		for _, file := range fileList {
			fileExpStr, _ := core.NewExpString(file)
			err := api.ExpDiaDB_AddFileList3(fileExpStr, ioLevel, true, diadbFileList)
			if err != core.EXP_SUCCESS {
				panic(err)
			}
		}
	}

	var err core.ExpErr
	diadbHandler := api.ExpDiaDB_Initiate(diadbFileList, 0, &err)
	if err != core.EXP_SUCCESS {
		panic(err)
	}

	api.ExpDiaDB_DeleteFileList(diadbFileList)

	return diadbHandler
}

// 鉄道ダイヤデータの終了処理
func termDiaData(diadbHandler core.ExpDiaDBHandler) {
	api.ExpDiaDB_Terminate(diadbHandler)
}

// KNBデータの初期処理
func initKnbData(knbDir string, diadbHandler core.ExpDiaDBHandler, addFile core.ExpString) core.ExpDataHandler {
	fileList := getEntryFileList(knbDir, ".knb")
	if len(fileList) == 0 && addFile == nil {
		return nil
	}

	var err core.ExpErr

	var knbFileList core.ExpDataFileList = api.ExpDB_NewFileList()
	if knbFileList != nil {
		var ioLevel core.ExpInt = core.EXP_IO_LEVEL_WIN32_AND_MMAP

		for _, file := range fileList {
			var fileId core.ExpFileID
			fileExpStr, _ := core.NewExpString(file)
			err = api.ExpDB_AddFileList3(fileExpStr, ioLevel, true, knbFileList, &fileId)
			if err != core.EXP_SUCCESS {
				panic(err)
			}
		}

		if addFile != nil {
			var fileId core.ExpFileID
			err = api.ExpDB_AddFileList3(addFile, ioLevel, true, knbFileList, &fileId)
			if err != core.EXP_SUCCESS {
				panic(err)
			}
		}
	}

	// ダイヤDB初期処理
	var knbHandler core.ExpDataHandler = api.ExpDB_Initiate3(knbFileList, diadbHandler, true, true, 2, &err)
	if err != core.EXP_SUCCESS {
		panic(err)
	}

	api.ExpDB_DeleteFileList(knbFileList)

	return knbHandler
}

// KNBデータの終了処理
func termKnbData(knbHandler core.ExpDataHandler) {
	api.ExpDB_Terminate(knbHandler)
}

func main() {
	//fmt.Println("Start.")

	// パラメータ解析
	flag.Parse()

	//fmt.Println("-k:" + *knbdir)
	//fmt.Println("-d:" + *diadir)
	//fmt.Println("-o:" + *outfile)
	//fmt.Println("-v:" + *version)
	//fmt.Println("-t:" + *idtable)

	//diaFileList := getEntryFileList(*diadir, ".dat")
	//fmt.Println(diaFileList)

	// 定数の値を出力
	fmt.Println((DISPLINE_MST_ID))

	if outfile != nil {
		// パス内のスラッシュを円マークに置換
		outfile2 := replacePath(*outfile)
		//fmt.Println((outfile2))
		outfile3 := outfile2 + ".copy"
		outfile2ExpStr, _ := core.NewExpString(outfile2)

		if idtable != nil {
			if FileExists(*idtable) {
				var mst string
				var ptn string

				// idtable.txt読込
				lines := readTextFile(*idtable)

				for i := 0; i < len(lines); i++ {
					// 文字数が0より大きいか判定
					if utf8.RuneCountInString(lines[i]) > 0 {
						//fmt.Println(strconv.Itoa(i) + "行目:" + lines[i])

						// カンマで文字列を分割し配列に格納
						var items = strings.Split(lines[i], ",")
						//fmt.Println("1番目:" + items[0])
						//fmt.Println("2番目:" + items[1])

						blockFile := filepath.Dir(*idtable) + "\\" + items[1]

						no, _ := strconv.Atoi(items[0])
						if FileExists(blockFile) {
							// KNBファイルの指定されたIDのデータブロックを取り除く
							api.ExpKNBFile_RemoveDataBlock(core.ExpInt32(no), outfile2ExpStr)
							//fmt.Println("ExpKNBFile_RemoveDataBlock")
							//fmt.Println(ans)

							// ファイルで指定されたデータブロックを任意のIDでKNBファイルに追加する
							blockFileExpStr, _ := core.NewExpString(blockFile)
							api.ExpKNBFile_AppendDataBlock(blockFileExpStr, 0, 0, core.ExpInt32(no), outfile2ExpStr)
							//fmt.Println("ExpKNBFile_AppendDataBlock")
							//fmt.Println(ans)

							if no == DISPLINE_MST_ID {
								mst = blockFile
							} else {
								ptn = blockFile
							}

							if mst != "" && ptn != "" {
								// ファイルのコピーを作成
								FileCopy(outfile2, outfile3)

								outfile3ExpStr, _ := core.NewExpString(outfile3)
								api.ExpKNBFile_RemoveDataBlock(DISPLINE_TRN_ID, outfile3ExpStr)
								//fmt.Println("ExpKNBFile_RemoveDataBlock")
								//fmt.Println(ans)

								diaHandler := initDiaData(*diadir)
								if diaHandler != nil {
									knbHandler := initKnbData(*knbdir, diaHandler, outfile3ExpStr)
									if knbHandler != nil {

										//diadb := api.ExpDLineData_Initiate(knbHandler, diaHandler, ????, ????, &err)
										// if diadb != nil {
										api.ExpDB_DLineBridge(knbHandler)
										//fmt.Println("ExpDB_DLineBridge")
										//fmt.Println(ans)

										trn := filepath.Dir(*idtable) + "\\" + DISPLINE_TRN_FILE
										trnExpStr, _ := core.NewExpString(trn)
										api.ExpDeleteFile(trnExpStr)
										api.ExpKNBData_WriteTrainDispLinePtnDataToFile(knbHandler, trnExpStr)
										//fmt.Println("ExpKNBData_WriteTrainDispLinePtnDataToFile")
										//fmt.Println(ans)

										//api.ExpDLineData_Terminate(diadb)

										api.ExpKNBFile_RemoveDataBlock(DISPLINE_TRN_ID, outfile2ExpStr)
										//fmt.Println("ExpKNBFile_RemoveDataBlock")
										//fmt.Println(ans)

										api.ExpKNBFile_AppendDataBlock(trnExpStr, 0, 0, DISPLINE_TRN_ID, outfile2ExpStr)
										//fmt.Println("ExpKNBFile_AppendDataBlock")
										//fmt.Println(ans)

										//}

										termKnbData(knbHandler)
									}

									termDiaData(diaHandler)
								}

								// ファイルを削除する
								api.ExpDeleteFile(outfile3ExpStr)
							}
						}
					}
				}
			}
		}

		// 共通KNBのファイルIDを設定
		var fileId core.ExpFileID
		fileId[0] = 1   // KNBファイルタイプ
		fileId[1] = 100 // 共通KNB(JPCOM.KNBのID)

		api.ExpKNBFile_ReplaceFileID(outfile2ExpStr, (*core.ExpFileID)(&fileId))
		//fmt.Println("ExpKNBFile_ReplaceFileID")
		//fmt.Println(ans)

		// 日付文字列を数値に変換
		date, err := strconv.Atoi(*version)
		if err != nil {
			panic(err)
		}

		if *version != "" {
			// バージョンを設定
			api.ExpKNBFile_ReplaceDateVersion(outfile2ExpStr, core.ExpDate(date))
			//fmt.Println("ExpKNBFile_ReplaceDateVersion")
			//fmt.Println(ans)
		}
	}
}

// テキストファイルを１行ずつ読み込む
func readTextFile(fileName string) []string {
	// ファイルをオープン
	fp, err := os.Open(fileName)
	if err != nil {
		panic(err)
	}
	defer fp.Close()

	// テキスト読込用スキャナ
	scanner := bufio.NewScanner(fp)

	// １行ずつ処理
	var lines []string
	for scanner.Scan() {
		lines = append(lines, string(scanner.Text()))
	}

	return lines
}

// 文字列内のスラッシュ(/)を円マークに置換
func replacePath(file string) string {
	replaced := strings.Replace(file, "/", "\\", -1)
	return replaced
}

// フォルダ内から指定された拡張子を持つファイルの一覧を取得
func getEntryFileList(dir string, ext string) []string {
	files, err := os.ReadDir(dir)
	if err != nil {
		panic(err)
	}

	var paths []string
	for _, file := range files {
		if file.IsDir() {
			continue
		}

		newExt := filepath.Ext(file.Name())
		if strings.ToLower(newExt) == ext {
			newFilePath := dir + "\\" + file.Name()
			paths = append(paths, newFilePath)
		}
	}

	return paths
}

// ファイルの存在チェック
func FileExists(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil
}

// ファイルをコピー
func FileCopy(strFile string, dstFile string) {
	src, err := os.Open(strFile)
	if err != nil {
		panic(err)
	}
	defer src.Close()

	dst, err := os.Create(dstFile)
	if err != nil {
		panic(err)
	}
	defer dst.Close()

	_, err = io.Copy(dst, src)
	if err != nil {
		panic(err)
	}
}
