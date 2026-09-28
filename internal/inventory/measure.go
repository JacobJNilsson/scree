package inventory

import (
	"go/ast"
	"go/scanner"
	"go/token"
	"strings"
)

// complexity returns 1 plus the decision points of spec 002 in a body, without nested function literals.
func complexity(body ast.Node) int {
	cc := 1
	ast.Inspect(body, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.FuncLit:
			return false
		case *ast.IfStmt, *ast.ForStmt, *ast.RangeStmt:
			cc++
		case *ast.CaseClause:
			// A nil list marks the default clause.
			if n.List != nil {
				cc++
			}
		case *ast.CommClause:
			if n.Comm != nil {
				cc++
			}
		case *ast.BinaryExpr:
			if n.Op == token.LAND || n.Op == token.LOR {
				cc++
			}
		}
		return true
	})
	return cc
}

// nesting returns the deepest chain of if, for, switch, and select statements below a node at the given depth.
func nesting(node ast.Node, depth int) int {
	deepest := depth
	ast.Inspect(node, func(n ast.Node) bool {
		if n == node {
			return true
		}
		switch n := n.(type) {
		case *ast.FuncLit:
			return false
		case *ast.IfStmt:
			deepest = max(deepest, ifNesting(n, depth))
			return false
		case *ast.ForStmt, *ast.RangeStmt, *ast.SwitchStmt, *ast.TypeSwitchStmt, *ast.SelectStmt:
			deepest = max(deepest, nesting(n, depth+1))
			return false
		}
		return true
	})
	return deepest
}

// ifNesting keeps an else-if chain at the depth of its first if.
func ifNesting(stmt *ast.IfStmt, depth int) int {
	deepest := depth + 1
	for _, part := range []ast.Node{stmt.Init, stmt.Cond, stmt.Body} {
		if part != nil {
			deepest = max(deepest, nesting(part, depth+1))
		}
	}
	switch els := stmt.Else.(type) {
	case *ast.IfStmt:
		deepest = max(deepest, ifNesting(els, depth))
	case *ast.BlockStmt:
		deepest = max(deepest, nesting(els, depth+1))
	}
	return deepest
}

// codeLines returns the lines of a file that hold at least one token that is not a comment.
func codeLines(file *token.File, src []byte) map[int]bool {
	lines := map[int]bool{}
	var s scanner.Scanner
	s.Init(file, src, nil, 0)
	for {
		pos, tok, lit := s.Scan()
		if tok == token.EOF {
			return lines
		}
		start := file.PositionFor(pos, false).Line
		end := start
		if tok == token.STRING {
			end += strings.Count(lit, "\n")
		}
		for line := start; line <= end; line++ {
			lines[line] = true
		}
	}
}
