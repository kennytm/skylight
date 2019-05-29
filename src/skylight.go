package main

import (
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"log"
	"os"
	"path"
	"sort"
	"strings"

	"github.com/fatih/astrewrite"
	"golang.org/x/tools/cover"
)

type codePos struct {
	line int
	col  int
}

func (cp *codePos) less(other *codePos) bool {
	return cp.line < other.line || cp.line == other.line && cp.col < other.col
}

func (cp *codePos) lessEq(other *codePos) bool {
	return cp.line < other.line || cp.line == other.line && cp.col <= other.col
}

func (cp *codePos) String() string {
	return fmt.Sprintf("%d:%d", cp.line, cp.col)
}

type codeRange struct {
	start codePos
	end   codePos
}

func makeCodeRangeFromBlock(block cover.ProfileBlock) codeRange {
	return codeRange{
		start: codePos{
			line: block.StartLine,
			col:  block.StartCol - 1, // allows covering the entire empty block
		},
		end: codePos{
			line: block.EndLine,
			col:  block.EndCol,
		},
	}
}

func makeCodeRangeFromNode(fs *token.FileSet, node ast.Node) codeRange {
	start := fs.Position(node.Pos())
	end := fs.Position(node.End())
	return codeRange{
		start: codePos{
			line: start.Line,
			col:  start.Column,
		},
		end: codePos{
			line: end.Line,
			col:  end.Column,
		},
	}
}

func (cr codeRange) String() string {
	return fmt.Sprintf("%s--%s", &cr.start, &cr.end)
}

func writeFile(outPath string, fs *token.FileSet, node ast.Node) {
	if err := os.MkdirAll(path.Dir(outPath), 0755); err != nil {
		log.Fatalf("cannot create directory for `%s`: %v", outPath, err)
	}
	file, err := os.Create(outPath)
	if err != nil {
		log.Fatalf("cannot create `%s`: %v", outPath, err)
	}
	defer file.Close()

	if err := printer.Fprint(file, fs, node); err != nil {
		log.Fatalf("cannot write to `%s`: %v", outPath, err)
	}
}

type findRangeResult int

const (
	findRangeResultOverlapping findRangeResult = iota
	findRangeResultContained
	findRangeResultNonOverlapping
)

func findRange(ranges []codeRange, cr *codeRange) findRangeResult {
	idx := sort.Search(len(ranges), func(i int) bool {
		return cr.end.lessEq(&ranges[i].end)
	})

	//  previous  found
	//    [---)   [--)
	//             [)    contained        (cr.start >= found.start)
	//           [--)    overlapping      (cr.end > found.start)
	//          [)       non-overlapping  (cr.start >= previous.end)
	//      [----)       overlapping

	if idx < len(ranges) {
		foundStart := ranges[idx].start
		switch {
		case foundStart.lessEq(&cr.start):
			return findRangeResultContained
		case foundStart.less(&cr.end):
			return findRangeResultOverlapping
		}
	}

	if idx == 0 || ranges[idx-1].end.lessEq(&cr.start) {
		return findRangeResultNonOverlapping
	}
	return findRangeResultOverlapping
}

func replaceStmt(fs *token.FileSet, funcIdent *ast.Ident, node ast.Stmt) ast.Stmt {
	start := fs.Position(node.Pos())
	msg := fmt.Sprintf(`"<[[SKYLIGHT]]> hit uncovered statement at %s"`, start)
	return &ast.BlockStmt{
		List: []ast.Stmt{
			&ast.ExprStmt{
				X: &ast.CallExpr{
					Fun: funcIdent,
					Args: []ast.Expr{
						&ast.BasicLit{
							Kind:  token.STRING,
							Value: msg,
						},
					},
				},
			},
			node,
		},
	}
}

func isDecl(stmt ast.Stmt) bool {
	switch s := stmt.(type) {
	case *ast.DeclStmt:
		return true
	case *ast.AssignStmt:
		return s.Tok == token.DEFINE
	default:
		return false
	}
}

func rewriteNode(ranges []codeRange, fs *token.FileSet, funcIdent *ast.Ident, skippedNodes map[ast.Node]struct{}, node ast.Node) (ast.Node, bool) {
	if node == nil {
		return nil, true
	}
	if _, ok := skippedNodes[node]; ok {
		delete(skippedNodes, node)
		return node, true
	}

	cr := makeCodeRangeFromNode(fs, node)
	switch findRange(ranges, &cr) {
	case findRangeResultNonOverlapping:
		return node, false
	case findRangeResultOverlapping:
		break
	case findRangeResultContained:
		if stmt, ok := node.(ast.Stmt); ok && !isDecl(stmt) {
			return replaceStmt(fs, funcIdent, stmt), false
		}
	}

	switch s := node.(type) {
	case *ast.ForStmt:
		if isDecl(s.Init) {
			skippedNodes[s.Init] = struct{}{}
		}
		skippedNodes[s.Post] = struct{}{}
	case *ast.TypeSwitchStmt:
		if isDecl(s.Init) {
			skippedNodes[s.Init] = struct{}{}
		}
		skippedNodes[s.Assign] = struct{}{}
	case *ast.SwitchStmt:
		if isDecl(s.Init) {
			skippedNodes[s.Init] = struct{}{}
		}
	case *ast.IfStmt:
		if isDecl(s.Init) {
			skippedNodes[s.Init] = struct{}{}
		}
	}
	return node, true

}

func main() {
	covProfPath := flag.String("c", "", "the coverage profile `path` generated from go test")
	moduleName := flag.String("m", "", "the name of the covered module (e.g. github.com/user/repo)")
	srcDir := flag.String("i", "", "directory containing the covered Go source")
	outDir := flag.String("o", ".", "output directory")
	funcName := flag.String("f", "panic", "function name to call before the uncovered statements")
	flag.Parse()

	profs, err := cover.ParseProfiles(*covProfPath)
	if err != nil {
		log.Fatalf("failed to parse coverage profile `%s`: %v", *covProfPath, err)
	}

	funcIdent := ast.NewIdent(*funcName)
	fs := token.NewFileSet()
	for _, prof := range profs {
		ranges := make([]codeRange, 0, len(prof.Blocks))
		for _, block := range prof.Blocks {
			if block.Count == 0 {
				newRange := makeCodeRangeFromBlock(block)
				if len(ranges) > 0 && newRange.start.lessEq(&ranges[len(ranges)-1].end) {
					ranges[len(ranges)-1].end = newRange.end
				} else {
					ranges = append(ranges, newRange)
				}
			}
		}
		if len(ranges) == 0 {
			// skip 100% covered files.
			continue
		}

		origFileName := prof.FileName
		if !strings.HasPrefix(origFileName, *moduleName) {
			log.Fatalf("unknown file `%s` outside of module `%s`", origFileName, *moduleName)
		}
		stem := origFileName[len(*moduleName):]
		srcPath := path.Join(*srcDir, stem)

		log.Printf("processing `%s` with %d uncovered ranges...", stem, len(ranges))

		node, err := parser.ParseFile(fs, srcPath, nil, 0)
		if err != nil {
			log.Fatalf("cannot parse `%s`: %v", srcPath, err)
		}

		skippedNodes := make(map[ast.Node]struct{})
		newNode := astrewrite.Walk(node, func(n ast.Node) (ast.Node, bool) {
			return rewriteNode(ranges, fs, funcIdent, skippedNodes, n)
		})

		outPath := path.Join(*outDir, stem)
		writeFile(outPath, fs, newNode)
	}
}
