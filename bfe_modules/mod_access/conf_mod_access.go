// Copyright (c) 2019 Baidu, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package mod_access

import (
	"fmt"
)

import (
	"github.com/baidu/go-lib/log/log4go"
	gcfg "gopkg.in/gcfg.v1"
)

// ConfModAccess holds the config of access module.
type ConfModAccess struct {
	Log struct {
		LogPrefix   string // log file prefix
		LogDir      string // log file dir
		RotateWhen  string // rotate time
		BackupCount int    // log file backup number
	}

	Template struct {
		RequestTemplate string // access log formate string
		SessionTemplate string // session finish log formate string
	}
}

// ConfLoad loads config of access module from file.
func ConfLoad(filePath string) (*ConfModAccess, error) {
	var err error
	var cfg ConfModAccess

	err = gcfg.ReadFileInto(&cfg, filePath)
	if err != nil {
		return &cfg, err
	}

	err = cfg.Check()
	if err != nil {
		return &cfg, err
	}

	return &cfg, nil
}

func (cfg *ConfModAccess) Check() error {
	if cfg.Log.LogPrefix == "" {
		return fmt.Errorf("ModAccess.LogPrefix is empty")
	}

	if cfg.Log.LogDir == "" {
		return fmt.Errorf("ModAccess.LogDir is empty")
	}

	if !log4go.WhenIsValid(cfg.Log.RotateWhen) {
		return fmt.Errorf("ModAccess.RotateWhen invalid: %s", cfg.Log.RotateWhen)
	}

	if cfg.Log.BackupCount <= 0 {
		return fmt.Errorf("ModAccess.BackupCount should > 0: %d", cfg.Log.BackupCount)
	}

	if cfg.Template.RequestTemplate == "" {
		return fmt.Errorf("ModAccess.RequestTemplate not set")
	}

	if cfg.Template.SessionTemplate == "" {
		return fmt.Errorf("ModAccess.SessionTemplate not set")
	}

	return nil
}

func checkLogFmt(item LogFmtItem, logFmtType string) error {
	if logFmtType != Request && logFmtType != Session {
		return fmt.Errorf("logFmtType should be Request or Session")
	}

	domain, found := fmtItemDomainTable[item.Type]
	if !found {
		return fmt.Errorf("type : (%d, %s) not configured in domain table",
			item.Type, item.Key)
	}

	if domain != DomainAll && domain != logFmtType {
		return fmt.Errorf("type : (%d, %s) should not in request finish log",
			item.Type, item.Key)
	}

	return nil
}

func tokenTypeGet(templatePtr *string, offset int) (int, int, error) {
	templateLen := len(*templatePtr)

	for key, logItemType := range fmtTable {
		n := len(key)
		if offset+n > templateLen {
			continue
		}

		if key == (*templatePtr)[offset:(offset+n)] {
			return logItemType, offset + n - 1, nil
		}
	}

	return -1, -1, fmt.Errorf("no such log item format type : %s", *templatePtr)
}

func parseBracketToken(templatePtr *string, offset int) (LogFmtItem, int, error) {
	length := len(*templatePtr)

	var endOfBracket int
	for endOfBracket = offset + 1; endOfBracket < length; endOfBracket++ {
		if (*templatePtr)[endOfBracket] == '}' {
			break
		}
	}

	if endOfBracket >= length {
		return LogFmtItem{}, -1, fmt.Errorf("log format: { must be terminated by a }")
	}

	if endOfBracket == (length - 1) {
		return LogFmtItem{}, -1, fmt.Errorf("log format: } must followed a charactor")
	}

	key := (*templatePtr)[offset+1 : endOfBracket]

	logItemType, end, err := tokenTypeGet(templatePtr, endOfBracket+1)
	if err != nil {
		return LogFmtItem{}, -1, err
	}

	return LogFmtItem{key, logItemType}, end, nil
}

func parseLogTemplate(logTemplate string) ([]LogFmtItem, error) {
	reqFmts := []LogFmtItem{}

	start := 0
	templateLen := len(logTemplate)
	var token string

	for i := 0; i < templateLen; i++ {
		if logTemplate[i] != '$' {
			continue
		}

		if (i + 1) == templateLen {
			return nil, fmt.Errorf("log format: $ must followed with a charactor")
		}

		if start <= (i - 1) {
			token = logTemplate[start:i]
			item := LogFmtItem{token, FormatString}
			reqFmts = append(reqFmts, item)
		}

		if logTemplate[i+1] == '{' {
			item, end, err := parseBracketToken(&logTemplate, i+1)
			if err != nil {
				return nil, err
			}
			reqFmts = append(reqFmts, item)
			i = end
			start = end + 1

		} else {
			logItemType, end, err := tokenTypeGet(&logTemplate, i+1)
			if err != nil {
				return nil, err
			}

			token = logTemplate[(i + 1) : end+1]
			item := LogFmtItem{token, logItemType}
			reqFmts = append(reqFmts, item)

			i = end
			start = end + 1
		}
	}

	if start < templateLen {
		token = logTemplate[start:templateLen]
		item := LogFmtItem{token, FormatString}
		reqFmts = append(reqFmts, item)
	}

	return reqFmts, nil
}
