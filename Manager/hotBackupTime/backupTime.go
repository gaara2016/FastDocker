package main

import (
	"bufio"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
)

const BACKUP_TIME = "hotBackupWithBolbs"
const RESULT = "backup_result"

/*
	读取磁盘文件，按照镜像名字统计各blob备份的时间，最后计算出备份整个镜像需要的时间
*/
func main() {
	// 1. 读取各blobs备份时间
	file, err := os.Open(BACKUP_TIME)
	if err != nil {
		log.Println(err)
		os.Create(BACKUP_TIME)
	}
	defer file.Close()

	result, er := os.OpenFile(RESULT, os.O_TRUNC|os.O_WRONLY, 0644)
	if er != nil {
		log.Println(er)
		os.Create(RESULT)
	}
	defer result.Close()

	reader := bufio.NewReader(file)
	writer := bufio.NewWriter(result)
	var backupTime = make(map[string]float64, 30)
	for {
		line, err := reader.ReadString('\n')
		// 去掉换行回车符
		line = strings.TrimSuffix(line, "\n")
		line = strings.TrimSuffix(line, "\r")

		put := func(content string) {
			var time = 0.0
			content = strings.Replace(content, " ", "", -1)

			tmp := strings.Split(content, ":")
			name := tmp[0]
			if strings.HasSuffix(tmp[1], "ms") {
				timeWithMs := strings.Split(tmp[1], "ms")
				ms, _ := strconv.ParseFloat(timeWithMs[0], 64)
				time = time + ms
			} else {
				timeWithS := strings.Split(tmp[1], "s")
				s, _ := strconv.ParseFloat(timeWithS[0], 64)
				time = time + s*1000
			}
			_, ok := backupTime[name]
			if ok {
				backupTime[name] = backupTime[name] + time
			} else {
				backupTime[name] = time
			}
		}

		if err != nil || err == io.EOF {
			//最后一行
			if len(line) > 0 {
				put(line)
			}
			break
		}
		put(line)
	}

	for k, v := range backupTime {
		writer.WriteString(k + ":" + strconv.FormatFloat(v, 'f', 2, 64) + "ms\n")
		writer.Flush()
	}
}
