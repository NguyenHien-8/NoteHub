#include "IpcClient.h"
#include "MainWindow.h"

#include <QApplication>
#include <QCommandLineOption>
#include <QCommandLineParser>
#include <QDir>
#include <QFileInfo>
#include <QIcon>
#include <QMessageBox>
#include <QProcessEnvironment>
#include <QTextStream>
#include <QTimer>

#ifndef NOTEHUB_VERSION
#define NOTEHUB_VERSION "0.3.0"
#endif

namespace {
QString defaultBackendPath() {
    const QString overridePath = qEnvironmentVariable("NOTEHUB_BACKEND");
    if (!overridePath.trimmed().isEmpty()) return QFileInfo(overridePath).absoluteFilePath();
#ifdef Q_OS_WIN
    const QString name = "notehub-backend.exe";
#else
    const QString name = "notehub-backend";
#endif
    return QDir(QCoreApplication::applicationDirPath()).filePath(name);
}
}

int main(int argc, char *argv[]) {
    QApplication app(argc, argv);
    QCoreApplication::setApplicationName("NoteHub");
    QCoreApplication::setOrganizationName("NoteHub");
    QCoreApplication::setApplicationVersion(NOTEHUB_VERSION);
    app.setWindowIcon(QIcon(":/icons/NoteHub.png"));

    QCommandLineParser parser;
    parser.setApplicationDescription("NoteHub - Qt desktop frontend + Go backend");
    parser.addHelpOption();
    parser.addVersionOption();
    QCommandLineOption dataDir({"d", "data-dir"}, "Use a separate NoteHub data directory", "path");
    QCommandLineOption backendOpt("backend", "Path to notehub-backend executable", "path");
    parser.addOption(dataDir);
    parser.addOption(backendOpt);
    parser.process(app);

    const QString backendPath = parser.isSet(backendOpt) ? QFileInfo(parser.value(backendOpt)).absoluteFilePath() : defaultBackendPath();
    if (!QFileInfo::exists(backendPath)) {
        QMessageBox::critical(nullptr, QObject::tr("Thiếu Go backend"),
                              QObject::tr("Không tìm thấy backend:\n%1\n\nHãy chạy script build để NoteHub.exe và notehub-backend nằm cùng thư mục.").arg(backendPath));
        return 2;
    }

    IpcClient ipc;
    MainWindow window(&ipc);
    QStringList backendArgs;
    if (parser.isSet(dataDir)) backendArgs << "--data-dir" << parser.value(dataDir);

    bool becameReady = false;
    QObject::connect(&ipc, &IpcClient::ready, &app, [&](const QJsonObject &info) {
        becameReady = true;
        const int protocol = info.value("protocolVersion").toInt();
        if (protocol != 1) {
            QMessageBox::critical(nullptr, QObject::tr("IPC không tương thích"), QObject::tr("GUI cần protocol 1 nhưng backend trả về %1.").arg(protocol));
            app.quit();
            return;
        }
        window.initialize(info);
        window.showForDesktop();
    });
    QObject::connect(&ipc, &IpcClient::fatal, &app, [&](const QString &message) {
        QMessageBox::critical(nullptr, QObject::tr("NoteHub không thể khởi động"), message);
        app.quit();
    });
    QObject::connect(&ipc, &IpcClient::backendExited, &app, [&](int code, QProcess::ExitStatus status) {
        if (!becameReady) return;
        if (window.isVisible() && status == QProcess::CrashExit) {
            QMessageBox::critical(&window, QObject::tr("Go backend đã dừng"), QObject::tr("Backend kết thúc bất thường (mã %1). NoteHub sẽ đóng để bảo vệ trạng thái dữ liệu.").arg(code));
            window.close();
        }
    });

    ipc.startBackend(backendPath, backendArgs);
    QTimer::singleShot(10000, &app, [&] {
        if (!becameReady && ipc.isRunning()) {
            QMessageBox::critical(nullptr, QObject::tr("Backend phản hồi chậm"), QObject::tr("Go backend không gửi tín hiệu sẵn sàng sau 10 giây."));
            ipc.stop();
            app.quit();
        }
    });

    return app.exec();
}
