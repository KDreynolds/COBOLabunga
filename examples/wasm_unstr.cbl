       IDENTIFICATION DIVISION.
       PROGRAM-ID. wasmustr.
       DATA DIVISION.
       WORKING-STORAGE SECTION.
       01 WS-SOURCE PIC X(20) VALUE "HELLO WORLD".
       01 WS-WORD1  PIC X(10).
       01 WS-WORD2  PIC X(10).
       PROCEDURE DIVISION.
       MAIN.
           UNSTRING WS-SOURCE
               DELIMITED BY SPACE
               INTO WS-WORD1
                    WS-WORD2
           END-UNSTRING
           DISPLAY WS-WORD1
           DISPLAY WS-WORD2
           STOP RUN.
