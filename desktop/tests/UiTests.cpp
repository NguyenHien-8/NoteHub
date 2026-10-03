#include "BackendClient.h"
#include "ImageGallery.h"
#include "MainWindow.h"
#include "NoteEditor.h"
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
};
QTEST_MAIN(UiTests)
#include "UiTests.moc"
