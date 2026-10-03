#include "Theme.h"

#include <QApplication>
#include <QFont>
#include <QPalette>
#include <QStyleFactory>
#include <QStyleHints>

namespace {
bool systemDark(const QApplication &app) {
    return app.styleHints()->colorScheme() == Qt::ColorScheme::Dark;
}
}

QString Theme::normalizedMode(const QString &mode) {
    const QString value = mode.trimmed().toLower();
    if (value == "light" || value == "dark") return value;
    return "system";
}

void Theme::apply(QApplication &app, const QString &mode, const QString &fontFamily, int fontSize) {
    const QString normalized = normalizedMode(mode);
    const bool dark = normalized == "dark" || (normalized == "system" && systemDark(app));

    app.setStyle(QStyleFactory::create("Fusion"));
    QPalette palette;
    if (dark) {
        palette.setColor(QPalette::Window, QColor("#16191d"));
        palette.setColor(QPalette::WindowText, QColor("#e8edf2"));
        palette.setColor(QPalette::Base, QColor("#111418"));
        palette.setColor(QPalette::AlternateBase, QColor("#1d2228"));
        palette.setColor(QPalette::ToolTipBase, QColor("#242a31"));
        palette.setColor(QPalette::ToolTipText, QColor("#f7f9fb"));
        palette.setColor(QPalette::Text, QColor("#e8edf2"));
        palette.setColor(QPalette::Button, QColor("#242a31"));
        palette.setColor(QPalette::ButtonText, QColor("#e8edf2"));
        palette.setColor(QPalette::BrightText, Qt::red);
        palette.setColor(QPalette::Highlight, QColor("#5f8cff"));
        palette.setColor(QPalette::HighlightedText, QColor("#ffffff"));
        palette.setColor(QPalette::PlaceholderText, QColor("#8b97a5"));
    } else {
        palette.setColor(QPalette::Window, QColor("#f5f7fa"));
        palette.setColor(QPalette::WindowText, QColor("#18212b"));
        palette.setColor(QPalette::Base, QColor("#ffffff"));
        palette.setColor(QPalette::AlternateBase, QColor("#f0f3f7"));
        palette.setColor(QPalette::ToolTipBase, QColor("#ffffff"));
        palette.setColor(QPalette::ToolTipText, QColor("#18212b"));
        palette.setColor(QPalette::Text, QColor("#18212b"));
        palette.setColor(QPalette::Button, QColor("#ffffff"));
        palette.setColor(QPalette::ButtonText, QColor("#18212b"));
        palette.setColor(QPalette::BrightText, Qt::red);
        palette.setColor(QPalette::Highlight, QColor("#3f6fdd"));
        palette.setColor(QPalette::HighlightedText, QColor("#ffffff"));
        palette.setColor(QPalette::PlaceholderText, QColor("#7c8794"));
    }
    app.setPalette(palette);

    QFont font = app.font();
    if (!fontFamily.trimmed().isEmpty() && fontFamily != "System") font.setFamily(fontFamily);
    font.setPointSize(qBound(8, fontSize, 24));
    app.setFont(font);

    const QString bg = dark ? "#16191d" : "#f5f7fa";
    const QString panel = dark ? "#1d2228" : "#ffffff";
    const QString panelAlt = dark ? "#20262d" : "#f7f9fc";
    const QString border = dark ? "#303842" : "#dce3ea";
    const QString text = dark ? "#e8edf2" : "#18212b";
    const QString muted = dark ? "#99a6b5" : "#667587";
    const QString accent = dark ? "#6f96ff" : "#3f6fdd";
    const QString hover = dark ? "#2b333c" : "#edf3ff";
    const QString danger = dark ? "#ff7474" : "#c73d3d";

    app.setStyleSheet(QString(R"QSS(
        QMainWindow, QWidget#root { background: %1; color: %5; }
        QFrame#panel, QFrame#card, QFrame#composer, QFrame#rightPanel, QFrame#leftPanel {
            background: %2; border: 1px solid %4; border-radius: 12px;
        }
        QFrame#leftPanel, QFrame#rightPanel { background: %3; }
        QLabel#brand { font-size: 19px; font-weight: 700; }
        QLabel#sectionTitle { font-size: 17px; font-weight: 700; }
        QLabel#muted, QLabel#timestamp { color: %6; }
        QPushButton, QToolButton {
            min-height: 30px; padding: 5px 10px; border: 1px solid %4;
            border-radius: 8px; background: %2; color: %5;
        }
        QPushButton:hover, QToolButton:hover { background: %8; }
        QPushButton:checked, QToolButton:checked {
            background: %8; border-color: %7; color: %7; font-weight: 600;
        }
        QPushButton#primary { background: %7; color: white; border-color: %7; font-weight: 600; }
        QPushButton#danger { color: %9; }
        QLineEdit, QTextEdit, QPlainTextEdit, QComboBox, QSpinBox, QDateEdit, QListWidget, QTreeWidget {
            background: %2; color: %5; border: 1px solid %4; border-radius: 8px;
            padding: 6px; selection-background-color: %7;
        }
        QTextBrowser { background: transparent; border: none; color: %5; }
        QScrollArea { border: none; background: transparent; }
        QScrollArea > QWidget > QWidget { background: transparent; }
        QHeaderView::section { background: %3; color: %5; padding: 7px; border: none; border-bottom: 1px solid %4; }
        QCalendarWidget QWidget { alternate-background-color: %3; }
        QMenu { background: %2; color: %5; border: 1px solid %4; }
        QMenu::item:selected { background: %8; }
        QTabWidget::pane { border: 1px solid %4; border-radius: 8px; }
        QTabBar::tab { background: %3; padding: 8px 13px; margin-right: 2px; border-radius: 7px; }
        QTabBar::tab:selected { background: %8; color: %7; }
        QSplitter::handle { background: transparent; width: 8px; }
        QToolTip { background: %2; color: %5; border: 1px solid %4; padding: 5px; }
    )QSS")
        .arg(bg)
        .arg(panel)
        .arg(panelAlt)
        .arg(border)
        .arg(text)
        .arg(muted)
        .arg(accent)
        .arg(hover)
        .arg(danger));
}
