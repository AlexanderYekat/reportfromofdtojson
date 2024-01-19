sub getHyperlink()    
    q = Strings.Replace(v, "=HYPERLINK(""", "")
    q = Strings.Replace(q, """,""Перейти"")", "")
end sub
sub goCells()
    For i = 10 To 10000
        if Cells(i, 31).Value = "" then
         exit
        edn if
        v = Cells(i, 31).Formula
        h = getHyperlink(v)
        Cells(i, 27) = h
    Next i
end sub