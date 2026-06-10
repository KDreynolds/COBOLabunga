       IDENTIFICATION DIVISION.
       PROGRAM-ID. varydemo.
       DATA DIVISION.
       WORKING-STORAGE SECTION.
       01 WS-I          PIC 9(5).
       PROCEDURE DIVISION.
       MAIN.
           PERFORM VARYING WS-I FROM 1 BY 1 UNTIL WS-I > 5
               DISPLAY WS-I
           END-PERFORM
           DISPLAY "Done"
           STOP RUN.
