#include "MainWindow.h"
#include <QApplication>
#include <QCommandLineParser>
#include <QDir>
#include <QFontDatabase>
#include <QIcon>
#include <QStyleFactory>
#include <QThreadPool>

int main(int argc, char **argv) {
    QApplication app(argc, argv);
    app.setApplicationName("NoteHub");
    app.setOrganizationName("NoteHub");
    app.setApplicationVersion(NOTEHUB_VERSION);
    app.setWindowIcon(QIcon(":/icons/NoteHub.png"));
    if (auto fusion = QStyleFactory::create("Fusion"))
        app.setStyle(fusion);
    const auto families = QFontDatabase::families();
    const QString preferred = families.contains("Segoe UI Variable Text") ? "Segoe UI Variable Text"
                              : families.contains("Segoe UI")              ? "Segoe UI"
                                                                           : app.font().family();
    app.setFont(QFont(preferred, 10));
    QThreadPool::globalInstance()->setMaxThreadCount(4);

    QCommandLineParser parser;
    parser.setApplicationDescription("NoteHub — Qt desktop with Go backend");
    parser.addHelpOption();
    parser.addVersionOption();
    parser.addOption({"data-dir", "Use a separate NoteHub data directory", "directory"});
    parser.addOption({"backend", "Use an explicit backend executable", "path"});
    parser.process(app);
    QString core = parser.value("backend");
    if (core.isEmpty())
        core = QDir(QCoreApplication::applicationDirPath())
                   .filePath(
#ifdef Q_OS_WIN
                       "notehub-core.exe"
#else
                       "notehub-core"
#endif
                   );
    MainWindow window(core, parser.value("data-dir"));
    window.showMaximized();
    return app.exec();
}
