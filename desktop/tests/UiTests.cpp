#include "BackendClient.h"
#include "ImageGallery.h"
#include "MainWindow.h"
#include "NoteEditor.h"
#include <QClipboard>
#include <QMimeData>
#include <QtTest>
#include <QtWidgets>

class UiTests final : public QObject {
    Q_OBJECT
  private slots:
    void editorPreservesUntouchedMarkdown() {
        NoteEditor editor;
        const QString original = "# Ghi chú\n\n**Bold** #Research/FPGA";
        editor.setContent(original);
        QCOMPARE(editor.content(), original);
        QVERIFY(!editor.modified());
        auto text = editor.textEdit();
        text->selectAll();
        QTextCharFormat format;
        format.setFontUnderline(true);
        format.setForeground(QColor("#ab1234"));
        text->mergeCurrentCharFormat(format);
        QVERIFY(editor.modified());
        QVERIFY(editor.content().startsWith(RichTextPrefix));
        NoteEditor reopened;
        reopened.setContent(editor.content());
        reopened.textEdit()->selectAll();
        QVERIFY(reopened.textEdit()->textCursor().charFormat().fontUnderline());
    }
    void editorPasteKeepsFormattingAndLineBreaks() {
        NoteEditor editor;
        editor.show();
        auto text = editor.textEdit();
        text->setFocus();

        auto mime = new QMimeData;
        mime->setText("first line\nsecond line\nthird line");
        mime->setHtml("<div><b>first line</b></div><div>second line</div><div>third line</div>");
        QGuiApplication::clipboard()->setMimeData(mime);
        text->paste();

        QCOMPARE(text->toPlainText(), QString("first line\nsecond line\nthird line"));
        const auto saved = editor.content();
        QVERIFY(saved.startsWith(RichTextPrefix));

        NoteEditor reopened;
        reopened.setContent(saved);
        QCOMPARE(reopened.textEdit()->toPlainText(), QString("first line\nsecond line\nthird line"));
        auto cursor = reopened.textEdit()->textCursor();
        cursor.setPosition(0);
        cursor.movePosition(QTextCursor::NextWord, QTextCursor::KeepAnchor);
        QVERIFY(cursor.charFormat().fontWeight() >= QFont::DemiBold);

        NoteEditor codeEditor;
        auto codeMime = new QMimeData;
        const QString command = ".\\scripts\\build-windows.ps1 `\n  -QtRoot \"C:\\Qt\\6.11.2\\mingw_64\" `\n  -Clean";
        codeMime->setText(command);
        QGuiApplication::clipboard()->setMimeData(codeMime);
        codeEditor.textEdit()->paste();
        NoteEditor codeReopened;
        codeReopened.setContent(codeEditor.content());
        QCOMPARE(codeReopened.textEdit()->toPlainText(), command);
    }

    void markdownAndRichTextDoNotLoadResources() {
        LocalTextDocument doc;
        loadNoteDocument(&doc, "![private](file:///missing/private.png)");
        QVERIFY(doc.resource(QTextDocument::ImageResource, QUrl("file:///missing/private.png")).isNull());
    }
    void galleryUsesWidthAndReflows() {
        QJsonArray values;
        for (int i = 0; i < 7; ++i)
            values.append(QJsonObject{
                {"uid", QString::number(i)}, {"filename", "sample.png"}, {"mimeType", "image/png"}});
        ImageGallery gallery(values);
        int wideHeight = gallery.heightForWidth(1400);
        QVERIFY(gallery.heightForWidth(360) > wideHeight);
        for (int width : {320, 900, 1400, 460}) {
            auto frames = gallery.frames(width);
            QCOMPARE(frames.size(), 7);
            for (const auto &r : frames) {
                QVERIFY(r.left() >= 0);
                QVERIFY(r.right() < width);
                QVERIFY(r.bottom() < gallery.heightForWidth(width));
            }
        }
        const auto wide = gallery.frames(1400);
        QCOMPARE(wide[4].width(), wide[0].width());
        QVERIFY(wide[4].width() < 400); // the incomplete last row must not become giant previews
    }
    void ipcHandlesFragmentedRepliesAndEOF() {
        BackendClient backend;
        QSignalSpy ready(&backend, &BackendClient::connected);
        QSignalSpy fail(&backend, &BackendClient::failed);
        QString helper = QCoreApplication::applicationDirPath() + "/notehub-fake-core"
#ifdef Q_OS_WIN
                                                                  ".exe"
#endif
            ;
        backend.start(helper, {});
        QTRY_COMPARE_WITH_TIMEOUT(ready.count(), 1, 10000);
        QVERIFY(backend.isReady());
        bool replied = false;
        backend.call(this, "echo", {{"text", QString::fromUtf8("Ghi chú 🧪")}},
                     [&](const QJsonValue &value, const QString &code, const QString &) {
                         QVERIFY(code.isEmpty());
                         QCOMPARE(value.toObject()["text"].toString(), QString::fromUtf8("Ghi chú 🧪"));
                         replied = true;
                     });
        QTRY_VERIFY(replied);
        QVERIFY(!backend.busy());
        backend.stop();
        QCOMPARE(fail.count(), 0);
    }
    void desktopStartsAndRendersWithIsolatedSettings() {
        QTemporaryDir settingsDir;
        QSettings::setPath(QSettings::IniFormat, QSettings::UserScope, settingsDir.path());
        QSettings::setDefaultFormat(QSettings::IniFormat);
        QString helper = QCoreApplication::applicationDirPath() + "/notehub-fake-core"
#ifdef Q_OS_WIN
                                                                  ".exe"
#endif
            ;
        MainWindow window(helper, {});
        window.show();
        QTest::qWait(200);
        window.resize(820, 650);
        QVERIFY(!window.grab().isNull());
        const auto output = qEnvironmentVariable("NOTEHUB_QT_SCREENSHOTS");
        if (!output.isEmpty()) {
            QDir().mkpath(output);
            window.grab().save(output + "/qt-home-narrow.png");
            window.resize(1450, 900);
            QTest::qWait(100);
            window.grab().save(output + "/qt-home-wide.png");
        }
        QTRY_VERIFY_WITH_TIMEOUT(window.findChild<BackendClient *>()->isReady(), 10000);
    }
    void sidebarSplitterCanResizeNavigation() {
        QTemporaryDir settingsDir;
        QSettings::setPath(QSettings::IniFormat, QSettings::UserScope, settingsDir.path());
        QSettings::setDefaultFormat(QSettings::IniFormat);
        QString helper = QCoreApplication::applicationDirPath() + "/notehub-fake-core"
#ifdef Q_OS_WIN
                                                                  ".exe"
#endif
            ;
        MainWindow window(helper, {});
        window.resize(1400, 860);
        window.show();
        QTRY_VERIFY_WITH_TIMEOUT(window.findChild<BackendClient *>()->isReady(), 10000);

        auto splitter = window.findChild<QSplitter *>("navigationSplitter");
        auto nav = window.findChild<QScrollArea *>("navScroll");
        QVERIFY(splitter);
        QVERIFY(nav);
        splitter->setSizes({220, 1100});
        QCoreApplication::processEvents();
        QVERIFY(nav->width() >= 190);
        splitter->setSizes({68, 1250});
        QCoreApplication::processEvents();
        QVERIFY(nav->width() <= 90);
    }

    void themeUpdatesCalendarAndSettingsStayCompact() {
        QTemporaryDir settingsDir;
        QSettings::setPath(QSettings::IniFormat, QSettings::UserScope, settingsDir.path());
        QSettings::setDefaultFormat(QSettings::IniFormat);

        QString helper = QCoreApplication::applicationDirPath() + "/notehub-fake-core"
#ifdef Q_OS_WIN
                                                                  ".exe"
#endif
            ;
        MainWindow window(helper, {});
        window.resize(1400, 860);
        window.show();
        QTRY_VERIFY_WITH_TIMEOUT(window.findChild<BackendClient *>()->isReady(), 10000);

        auto miniCalendar = window.findChild<QCalendarWidget *>("miniCalendar");
        QVERIFY(miniCalendar);

        QToolButton *settingsButton = nullptr;
        for (auto button : window.findChildren<QToolButton *>())
            if (button->property("label").toString() == "settings") {
                settingsButton = button;
                break;
            }
        QVERIFY(settingsButton);
        QTest::mouseClick(settingsButton, Qt::LeftButton);

        QTRY_VERIFY(window.findChild<QFontComboBox *>("fontCombo"));
        auto font = window.findChild<QFontComboBox *>("fontCombo");
        auto appearance = window.findChild<QComboBox *>("appearanceCombo");
        auto size = window.findChild<QSpinBox *>("fontSizeSpin");
        QVERIFY(font);
        QVERIFY(appearance);
        QVERIFY(size);
        QVERIFY(font->isEditable());
        QVERIFY(font->maximumWidth() <= 340);
        QVERIFY(appearance->maximumWidth() <= 230);
        QVERIFY(size->maximumWidth() <= 120);
        auto calendarView = miniCalendar->findChild<QAbstractItemView *>();
        QVERIFY(calendarView);
        appearance->setCurrentText("Dark");
        QCoreApplication::processEvents();
        QVERIFY(calendarView->viewport()->palette().color(QPalette::Base).lightness() < 128);

        appearance->setCurrentText("Light");
        QCoreApplication::processEvents();
        QVERIFY(calendarView->viewport()->palette().color(QPalette::Base).lightness() > 128);
    }
};
QTEST_MAIN(UiTests)
#include "UiTests.moc"
